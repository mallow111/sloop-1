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
	"reflect"
	"testing"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/salesforce/sloop/pkg/sloop/store/untyped"
	"github.com/salesforce/sloop/pkg/sloop/store/untyped/badgerwrap"
	"github.com/stretchr/testify/assert"
)

func helper_ResourceEventCounts_ShouldSkip() bool {
	// Tests will not work on the fake types in the template, but we want to run tests on real objects
	if "typed.Value"+"Type" == fmt.Sprint(reflect.TypeOf(ResourceEventCounts{})) {
		fmt.Printf("Skipping unit test")
		return true
	}
	return false
}

func Test_ResourceEventCountsTable_SetWorks(t *testing.T) {
	if helper_ResourceEventCounts_ShouldSkip() {
		return
	}

	untyped.TestHookSetPartitionDuration(time.Hour * 24)
	db, err := (&badgerwrap.MockFactory{}).Open(badger.DefaultOptions(""))
	assert.Nil(t, err)
	err = db.Update(func(txn badgerwrap.Txn) error {
		k := (&EventCountKey{}).GetTestKey()
		vt := OpenResourceEventCountsTable()
		err2 := vt.Set(txn, k, (&EventCountKey{}).GetTestValue())
		assert.Nil(t, err2)
		return nil
	})
	assert.Nil(t, err)
}

func helper_update_ResourceEventCountsTable(t *testing.T, keys []string, val *ResourceEventCounts) (badgerwrap.DB, *ResourceEventCountsTable) {
	b, err := (&badgerwrap.MockFactory{}).Open(badger.DefaultOptions(""))
	assert.Nil(t, err)
	wt := OpenResourceEventCountsTable()
	err = b.Update(func(txn badgerwrap.Txn) error {
		var txerr error
		for _, key := range keys {
			txerr = wt.Set(txn, key, val)
			if txerr != nil {
				return txerr
			}
		}
		// Add some keys outside the range
		txerr = txn.Set([]byte("/a/123/"), []byte{})
		if txerr != nil {
			return txerr
		}
		txerr = txn.Set([]byte("/zzz/123/"), []byte{})
		if txerr != nil {
			return txerr
		}
		return nil
	})
	assert.Nil(t, err)
	return b, wt
}

func Test_ResourceEventCountsTable_GetUniquePartitionList_Success(t *testing.T) {
	if helper_ResourceEventCounts_ShouldSkip() {
		return
	}

	db, wt := helper_update_ResourceEventCountsTable(t, (&EventCountKey{}).SetTestKeys(), (&EventCountKey{}).SetTestValue())
	var partList []string
	var err1 error
	err := db.View(func(txn badgerwrap.Txn) error {
		partList, err1 = wt.GetUniquePartitionList(txn)
		return nil
	})
	assert.Nil(t, err)
	assert.Nil(t, err1)
	assert.Len(t, partList, 3)
	assert.Contains(t, partList, someMinPartition)
	assert.Contains(t, partList, someMiddlePartition)
	assert.Contains(t, partList, someMaxPartition)
}

func Test_ResourceEventCountsTable_GetUniquePartitionList_EmptyPartition(t *testing.T) {
	if helper_ResourceEventCounts_ShouldSkip() {
		return
	}

	db, wt := helper_update_ResourceEventCountsTable(t, []string{}, (&EventCountKey{}).GetTestValue())
	var partList []string
	var err1 error
	err := db.View(func(txn badgerwrap.Txn) error {
		partList, err1 = wt.GetUniquePartitionList(txn)
		return err1
	})
	assert.Nil(t, err)
	assert.Len(t, partList, 0)
}

func Test_EventCount_GetPreviousKey_Success(t *testing.T) {
	db, wt := helper_update_ResourceEventCountsTable(t, (&EventCountKey{}).SetTestKeys(), (&EventCountKey{}).SetTestValue())
	var partRes *EventCountKey
	var err1 error
	curKey := NewEventCountKey(someMaxTs, someKind, someNamespace, someName, someUid+"c")
	keyComparator := NewEventCountKeyComparator(someKind, someNamespace, someName, someUid+"b")
	err := db.View(func(txn badgerwrap.Txn) error {
		partRes, err1 = wt.GetPreviousKey(txn, curKey, keyComparator)
		return err1
	})
	assert.Nil(t, err)
	expectedKey := NewEventCountKey(someMiddleTs, someKind, someNamespace, someName, someUid+"b")
	assert.Equal(t, expectedKey, partRes)
}

func Test_EventCount_GetPreviousKey_Fail(t *testing.T) {
	db, wt := helper_update_ResourceEventCountsTable(t, (&EventCountKey{}).SetTestKeys(), (&EventCountKey{}).SetTestValue())
	var partRes *EventCountKey
	var err1 error
	curKey := NewEventCountKey(someMaxTs, someKind, someNamespace, someName, someUid)
	keyComparator := NewEventCountKeyComparator(someKind+"b", someNamespace, someName, someUid)
	err := db.View(func(txn badgerwrap.Txn) error {
		partRes, err1 = wt.GetPreviousKey(txn, curKey, keyComparator)
		return err1
	})
	assert.NotNil(t, err)
	assert.Equal(t, &EventCountKey{}, partRes)
}
