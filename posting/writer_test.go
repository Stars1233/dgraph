/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package posting

import (
	"fmt"
	"math"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
	bpb "github.com/dgraph-io/badger/v4/pb"
)

var val = make([]byte, 128)

func BenchmarkWriter(b *testing.B) {
	createKVList := func() bpb.KVList {
		var KVList bpb.KVList
		for i := range 5000000 {
			n := &bpb.KV{Key: []byte(fmt.Sprint(i)), Value: val, Version: 5}
			KVList.Kv = append(KVList.Kv, n)
		}
		return KVList
	}

	// Creates separate writer for each thread
	writeInBadgerMThreadsB := func(db *badger.DB, KVList *bpb.KVList, wg *sync.WaitGroup) {
		defer wg.Done()
		wb := db.NewManagedWriteBatch()
		if err := wb.WriteList(KVList); err != nil {
			panic(err)
		}
		require.NoError(b, wb.Flush())

	}

	// Resuses one writer for all threads
	writeInBadgerMThreadsW := func(wb *badger.WriteBatch, KVList *bpb.KVList, wg *sync.WaitGroup) {
		defer wg.Done()

		if err := wb.WriteList(KVList); err != nil {
			panic(err)
		}

	}
	// Creates separate writer for each thread
	writeInBadgerSingleThreadB := func(db *badger.DB, KVList *bpb.KVList) {
		wb := db.NewManagedWriteBatch()
		if err := wb.WriteList(KVList); err != nil {
			panic(err)
		}
		require.NoError(b, wb.Flush())

	}
	// Resuses one writer for all threads
	writeInBadgerSingleThreadW := func(wb *badger.WriteBatch, KVList *bpb.KVList) {
		if err := wb.WriteList(KVList); err != nil {
			panic(err)
		}

	}

	dbOpts := badger.DefaultOptions("").
		WithLogger(nil).
		WithSyncWrites(false).
		WithNumVersionsToKeep(math.MaxInt64).
		WithCompression(options.None)

	KVList := createKVList()

	// Vanilla TxnWriter
	b.Run("TxnWriter", func(b *testing.B) {
		tmpIndexDir := b.TempDir()

		dbOpts.Dir = tmpIndexDir
		dbOpts.ValueDir = tmpIndexDir
		var db, err2 = badger.OpenManaged(dbOpts)
		require.NoError(b, err2)
		defer db.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			w := NewTxnWriter(db)
			for _, typ := range KVList.Kv {
				k := typ.Key
				v := typ.Value
				require.NoError(b, w.SetAt(k, v, BitSchemaPosting, 1))
			}
			require.NoError(b, w.Flush())

		}
	})
	// Single threaded BatchWriter
	b.Run("WriteBatch1", func(b *testing.B) {
		tmpIndexDir := b.TempDir()

		dbOpts.Dir = tmpIndexDir
		dbOpts.ValueDir = tmpIndexDir

		var db, err2 = badger.OpenManaged(dbOpts)
		require.NoError(b, err2)
		defer db.Close()

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			wb := db.NewManagedWriteBatch()
			if err := wb.WriteList(&KVList); err != nil {
				panic(err)
			}
			require.NoError(b, wb.Flush())
		}
	})
	// Multi threaded Batchwriter with thread contention in WriteBatch
	b.Run("WriteBatchMultThreadDiffWB", func(b *testing.B) {
		tmpIndexDir := b.TempDir()

		dbOpts.Dir = tmpIndexDir
		dbOpts.ValueDir = tmpIndexDir

		var db, err2 = badger.OpenManaged(dbOpts)
		require.NoError(b, err2)
		defer db.Close()

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			var wg sync.WaitGroup
			wg.Add(5)

			go writeInBadgerMThreadsB(db, &bpb.KVList{Kv: KVList.Kv[:1000000]}, &wg)
			go writeInBadgerMThreadsB(db, &bpb.KVList{Kv: KVList.Kv[1000001:2000000]}, &wg)
			go writeInBadgerMThreadsB(db, &bpb.KVList{Kv: KVList.Kv[2000001:3000000]}, &wg)
			go writeInBadgerMThreadsB(db, &bpb.KVList{Kv: KVList.Kv[3000001:4000000]}, &wg)
			go writeInBadgerMThreadsB(db, &bpb.KVList{Kv: KVList.Kv[4000001:]}, &wg)
			wg.Wait()

		}
	})
	// Multi threaded Batchwriter with thread contention in SetEntry
	b.Run("WriteBatchMultThreadSameWB", func(b *testing.B) {
		tmpIndexDir := b.TempDir()

		dbOpts.Dir = tmpIndexDir
		dbOpts.ValueDir = tmpIndexDir

		var db, err2 = badger.OpenManaged(dbOpts)
		require.NoError(b, err2)
		defer db.Close()

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			var wg sync.WaitGroup
			wg.Add(5)
			wb := db.NewManagedWriteBatch()
			go writeInBadgerMThreadsW(wb, &bpb.KVList{Kv: KVList.Kv[:1000000]}, &wg)
			go writeInBadgerMThreadsW(wb, &bpb.KVList{Kv: KVList.Kv[1000001:2000000]}, &wg)
			go writeInBadgerMThreadsW(wb, &bpb.KVList{Kv: KVList.Kv[2000001:3000000]}, &wg)
			go writeInBadgerMThreadsW(wb, &bpb.KVList{Kv: KVList.Kv[3000001:4000000]}, &wg)
			go writeInBadgerMThreadsW(wb, &bpb.KVList{Kv: KVList.Kv[4000001:]}, &wg)

			wg.Wait()
			require.NoError(b, wb.Flush())
		}
	})
	b.Run("WriteBatchSingleThreadDiffWB", func(b *testing.B) {
		tmpIndexDir := b.TempDir()

		dbOpts.Dir = tmpIndexDir
		dbOpts.ValueDir = tmpIndexDir

		var db, err2 = badger.OpenManaged(dbOpts)
		require.NoError(b, err2)
		defer db.Close()

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			writeInBadgerSingleThreadB(db, &bpb.KVList{Kv: KVList.Kv[:1000000]})
			writeInBadgerSingleThreadB(db, &bpb.KVList{Kv: KVList.Kv[1000001:2000000]})
			writeInBadgerSingleThreadB(db, &bpb.KVList{Kv: KVList.Kv[2000001:3000000]})
			writeInBadgerSingleThreadB(db, &bpb.KVList{Kv: KVList.Kv[3000001:4000000]})
			writeInBadgerSingleThreadB(db, &bpb.KVList{Kv: KVList.Kv[4000001:]})
		}
	})
	b.Run("WriteBatchSingleThreadSameWB", func(b *testing.B) {
		tmpIndexDir := b.TempDir()

		dbOpts.Dir = tmpIndexDir
		dbOpts.ValueDir = tmpIndexDir

		var db, err2 = badger.OpenManaged(dbOpts)
		require.NoError(b, err2)
		defer db.Close()

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			wb := db.NewManagedWriteBatch()
			writeInBadgerSingleThreadW(wb, &bpb.KVList{Kv: KVList.Kv[:1000000]})
			writeInBadgerSingleThreadW(wb, &bpb.KVList{Kv: KVList.Kv[1000001:2000000]})
			writeInBadgerSingleThreadW(wb, &bpb.KVList{Kv: KVList.Kv[2000001:3000000]})
			writeInBadgerSingleThreadW(wb, &bpb.KVList{Kv: KVList.Kv[3000001:4000000]})
			writeInBadgerSingleThreadW(wb, &bpb.KVList{Kv: KVList.Kv[4000001:]})
			require.NoError(b, wb.Flush())
		}
	})
}
