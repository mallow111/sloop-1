// This file was automatically generated by genny.
// Any changes will be lost if this file is regenerated.
// see https://github.com/cheekybits/genny

/*
 * Copyright (c) 2019, salesforce.com, inc.
 * All rights reserved.
 * SPDX-License-Identifier: BSD-3-Clause
 * For full license text, see LICENSE.txt file in the repo root or https://opensource.org/licenses/BSD-3-Clause
 */

package typed

import (
	"fmt"
	"strconv"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/salesforce/sloop/pkg/sloop/store/untyped"
	"github.com/salesforce/sloop/pkg/sloop/store/untyped/badgerwrap"
)

type ResourceEventCountsTable struct {
	tableName string
}

func OpenResourceEventCountsTable() *ResourceEventCountsTable {
	keyInst := &EventCountKey{}
	return &ResourceEventCountsTable{tableName: keyInst.TableName()}
}

func (t *ResourceEventCountsTable) Set(txn badgerwrap.Txn, key string, value *ResourceEventCounts) error {
	err := (&EventCountKey{}).ValidateKey(key)
	if err != nil {
		return errors.Wrapf(err, "invalid key for table %v: %v", t.tableName, key)
	}

	outb, err := proto.Marshal(value)
	if err != nil {
		return errors.Wrapf(err, "protobuf marshal for table %v failed", t.tableName)
	}

	err = txn.Set([]byte(key), outb)
	if err != nil {
		return errors.Wrapf(err, "set for table %v failed", t.tableName)
	}
	return nil
}

func (t *ResourceEventCountsTable) Get(txn badgerwrap.Txn, key string) (*ResourceEventCounts, error) {
	err := (&EventCountKey{}).ValidateKey(key)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid key for table %v: %v", t.tableName, key)
	}

	item, err := txn.Get([]byte(key))
	if err == badger.ErrKeyNotFound {
		// Dont wrap. Need to preserve error type
		return nil, err
	} else if err != nil {
		return nil, errors.Wrapf(err, "get failed for table %v", t.tableName)
	}

	valueBytes, err := item.ValueCopy([]byte{})
	if err != nil {
		return nil, errors.Wrapf(err, "value copy failed for table %v", t.tableName)
	}

	retValue := &ResourceEventCounts{}
	err = proto.Unmarshal(valueBytes, retValue)
	if err != nil {
		return nil, errors.Wrapf(err, "protobuf unmarshal failed for table %v on value length %v", t.tableName, len(valueBytes))
	}
	return retValue, nil
}

func (t *ResourceEventCountsTable) GetMinKey(txn badgerwrap.Txn) (bool, string) {
	keyPrefix := "/" + t.tableName + "/"
	iterOpt := badger.DefaultIteratorOptions
	iterOpt.Prefix = []byte(keyPrefix)
	iterator := txn.NewIterator(iterOpt)
	defer iterator.Close()
	iterator.Seek([]byte(keyPrefix))
	if !iterator.ValidForPrefix([]byte(keyPrefix)) {
		return false, ""
	}
	return true, string(iterator.Item().Key())
}

func (t *ResourceEventCountsTable) GetMaxKey(txn badgerwrap.Txn) (bool, string) {
	keyPrefix := "/" + t.tableName + "/"
	iterOpt := badger.DefaultIteratorOptions
	iterOpt.Prefix = []byte(keyPrefix)
	iterOpt.Reverse = true
	iterator := txn.NewIterator(iterOpt)
	defer iterator.Close()
	// We need to seek to the end of the range so we add a 255 character at the end
	iterator.Seek([]byte(keyPrefix + string(rune(255))))
	if !iterator.Valid() {
		return false, ""
	}
	return true, string(iterator.Item().Key())
}

func (t *ResourceEventCountsTable) GetMinMaxPartitions(txn badgerwrap.Txn) (bool, string, string) {
	ok, minKeyStr := t.GetMinKey(txn)
	if !ok {
		return false, "", ""
	}
	ok, maxKeyStr := t.GetMaxKey(txn)
	if !ok {
		// This should be impossible
		return false, "", ""
	}

	minKey := &EventCountKey{}
	maxKey := &EventCountKey{}

	err := minKey.Parse(minKeyStr)
	if err != nil {
		panic(fmt.Sprintf("invalid key in table: %v key: %q error: %v", t.tableName, minKeyStr, err))
	}

	err = maxKey.Parse(maxKeyStr)
	if err != nil {
		panic(fmt.Sprintf("invalid key in table: %v key: %q error: %v", t.tableName, maxKeyStr, err))
	}

	return true, minKey.PartitionId, maxKey.PartitionId
}

//TODO: will be replaced by GetPartitionsFromTimeRange in future
func (t *ResourceEventCountsTable) RangeRead(
	txn badgerwrap.Txn,
	keyPredicateFn func(string) bool,
	valPredicateFn func(*ResourceEventCounts) bool,
	startTime time.Time,
	endTime time.Time) (map[EventCountKey]*ResourceEventCounts, RangeReadStats, error) {

	resources := map[EventCountKey]*ResourceEventCounts{}

	keyPrefix := "/" + t.tableName + "/"
	iterOpt := badger.DefaultIteratorOptions
	iterOpt.Prefix = []byte(keyPrefix)
	itr := txn.NewIterator(iterOpt)
	defer itr.Close()

	startPartition := untyped.GetPartitionId(startTime)
	endPartition := untyped.GetPartitionId(endTime)
	startPartitionPrefix := keyPrefix + startPartition + "/"

	stats := RangeReadStats{}
	before := time.Now()

	lastPartition := ""
	for itr.Seek([]byte(startPartitionPrefix)); itr.ValidForPrefix([]byte(keyPrefix)); itr.Next() {
		stats.RowsVisitedCount += 1
		if !keyPredicateFn(string(itr.Item().Key())) {
			continue
		}
		stats.RowsPassedKeyPredicateCount += 1

		key := EventCountKey{}
		err := key.Parse(string(itr.Item().Key()))
		if err != nil {
			return nil, stats, err
		}
		if key.PartitionId != lastPartition {
			stats.PartitionCount += 1
			lastPartition = key.PartitionId
		}
		// partitions are zero padded to 12 digits so we can compare them lexicographically
		if key.PartitionId > endPartition {
			// end of range
			break
		}
		valueBytes, err := itr.Item().ValueCopy([]byte{})
		if err != nil {
			return nil, stats, err
		}
		retValue := &ResourceEventCounts{}
		err = proto.Unmarshal(valueBytes, retValue)
		if err != nil {
			return nil, stats, err
		}
		if valPredicateFn != nil && !valPredicateFn(retValue) {
			continue
		}
		stats.RowsPassedValuePredicateCount += 1
		resources[key] = retValue
	}
	stats.Elapsed = time.Since(before)
	stats.TableName = (&EventCountKey{}).TableName()
	return resources, stats, nil
}

//todo: need to add unit test
func (t *ResourceEventCountsTable) GetUniquePartitionList(txn badgerwrap.Txn) ([]string, error) {
	resources := []string{}
	ok, minPar, maxPar := t.GetMinMaxPartitions(txn)
	if ok {
		parDuration := untyped.GetPartitionDuration()
		for curPar := minPar; curPar <= maxPar; {
			resources = append(resources, curPar)
			// update curPar
			partInt, err := strconv.ParseInt(curPar, 10, 64)
			if err != nil {
				return resources, errors.Wrapf(err, "failed to get partition:%v", curPar)
			}
			parTime := time.Unix(partInt, 0).UTC().Add(parDuration)
			curPar = untyped.GetPartitionId(parTime)
		}
	}
	return resources, nil
}

//todo: need to add unit test
func (t *ResourceEventCountsTable) GetPreviousKey(txn badgerwrap.Txn, key *EventCountKey, keyPrefix *EventCountKey) (*EventCountKey, error) {
	partitionList, err := t.GetUniquePartitionList(txn)
	if err != nil {
		return &EventCountKey{}, errors.Wrapf(err, "failed to get partition list from table:%v", t.tableName)
	}
	currentPartition := key.PartitionId
	for i := len(partitionList) - 1; i >= 0; i-- {
		prePart := partitionList[i]
		if prePart > currentPartition {
			continue
		} else {
			prevFound, prevKey, err := t.getLastMatchingKeyInPartition(txn, prePart, key, keyPrefix)
			if err != nil {
				return &EventCountKey{}, errors.Wrapf(err, "Failure getting previous key for %v, for partition id:%v", key.String(), prePart)
			}
			if prevFound && err == nil {
				return prevKey, nil
			}
		}
	}
	return &EventCountKey{}, fmt.Errorf("failed to get any previous key in table:%v, for key:%v, keyPrefix:%v", t.tableName, key.String(), keyPrefix)
}

//todo: need to add unit test
func (t *ResourceEventCountsTable) getLastMatchingKeyInPartition(txn badgerwrap.Txn, curPartition string, curKey *EventCountKey, keyPrefix *EventCountKey) (bool, *EventCountKey, error) {
	iterOpt := badger.DefaultIteratorOptions
	iterOpt.Reverse = true
	itr := txn.NewIterator(iterOpt)
	defer itr.Close()

	oldKey := curKey.String()

	// update partition with current value
	curKey.SetPartitionId(curPartition)
	keySeekStr := curKey.String() + string(rune(255))

	itr.Seek([]byte(keySeekStr))

	// if the result is same as key, we want to check its previous one
	if oldKey == string(itr.Item().Key()) {
		itr.Next()
	}

	if itr.ValidForPrefix([]byte(keyPrefix.String())) {
		key := &EventCountKey{}
		err := key.Parse(string(itr.Item().Key()))
		if err != nil {
			return true, &EventCountKey{}, err
		}
		return true, key, nil
	}
	return false, &EventCountKey{}, nil
}

//todo: add unit tests
func (t *ResourceEventCountsTable) RangeReadPerPartition(txn badgerwrap.Txn, keyPrefix *EventCountKey,
	valPredicateFn func(*ResourceEventCounts) bool, startTime time.Time, endTime time.Time) (map[EventCountKey]*ResourceEventCounts, RangeReadStats, error) {
	resources := map[EventCountKey]*ResourceEventCounts{}

	stats := RangeReadStats{}
	before := time.Now()

	partitionList, err := t.GetPartitionsFromTimeRange(txn, startTime, endTime)
	if err != nil {
		return resources, stats, errors.Wrapf(err, "failed to get partitions from table:%v, from startTime:%v, to endTime:%v", t.tableName, startTime, endTime)
	}

	tablePrefix := "/" + t.tableName + "/"
	iterOpt := badger.DefaultIteratorOptions
	itr := txn.NewIterator(iterOpt)
	defer itr.Close()

	lastPartition := ""
	for _, currentPartition := range partitionList {
		curPartitionPrefix := tablePrefix + currentPartition + "/"
		itr.Seek([]byte(curPartitionPrefix))
		stats.RowsVisitedCount += 1

		// update keyPrefix's partition with current partition
		keyPrefix.SetPartitionId(currentPartition)
		prefixStr := keyPrefix.GetKeyPrefixString()
		if itr.ValidForPrefix([]byte(prefixStr)) {
			stats.RowsPassedKeyPredicateCount += 1
			key := EventCountKey{}
			err := key.Parse(string(itr.Item().Key()))
			if err != nil {
				return nil, stats, err
			}

			if key.PartitionId != lastPartition {
				stats.PartitionCount += 1
				lastPartition = key.PartitionId
			}

			valueBytes, err := itr.Item().ValueCopy([]byte{})
			if err != nil {
				return nil, stats, err
			}
			retValue := &ResourceEventCounts{}
			err = proto.Unmarshal(valueBytes, retValue)
			if err != nil {
				return nil, stats, err
			}
			if valPredicateFn != nil && !valPredicateFn(retValue) {
				continue
			}
			stats.RowsPassedValuePredicateCount += 1
			resources[key] = retValue
		}
	}

	stats.Elapsed = time.Since(before)
	stats.TableName = (&EventCountKey{}).TableName()
	return resources, stats, nil
}

//todo: need to add unit test
func (t *ResourceEventCountsTable) GetPartitionsFromTimeRange(txn badgerwrap.Txn, startTime time.Time, endTime time.Time) ([]string, error) {
	resources := []string{}
	startPartition := untyped.GetPartitionId(startTime)
	endPartition := untyped.GetPartitionId(endTime)
	parDuration := untyped.GetPartitionDuration()
	for curPar := startPartition; curPar <= endPartition; {
		resources = append(resources, curPar)
		// update curPar
		partInt, err := strconv.ParseInt(curPar, 10, 64)
		if err != nil {
			return resources, errors.Wrapf(err, "failed to get partition:%v", curPar)
		}
		parTime := time.Unix(partInt, 0).UTC().Add(parDuration)
		curPar = untyped.GetPartitionId(parTime)
	}
	return resources, nil
}
