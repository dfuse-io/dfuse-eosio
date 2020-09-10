package accounthist

//func Test_shardSummary(t *testing.T) {
//	kvStore, cleanup := getKVTestFactory(t)
//	defer cleanup()
//	maxEntries := uint64(10)
//
//	s := newTestService(kvStore, 0, 1000)
//	runShard(t, 0, maxEntries, kvStore,
//		ct.Block(t, "00000002bb",
//			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:cthing1", ct.GlobalSequence(3))),
//			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:cthing2", ct.GlobalSequence(4))),
//			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:cthing3", ct.GlobalSequence(5))),
//			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:cthing4", ct.GlobalSequence(6))),
//			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:cthing5", ct.GlobalSequence(7))),
//		),
//	)
//
//	runShard(t, 1, maxEntries, kvStore,
//		ct.Block(t, "00000001aa",
//			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:athing1", ct.GlobalSequence(1))),
//			ct.TrxTrace(t, ct.ActionTrace(t, "a:some:athing2", ct.GlobalSequence(2))),
//		),
//	)
//
//	summary, err := s.ShardSummary(context.Background(), eos.MustStringToName("a"))
//	require.NoError(t, err)
//	assert.Equal(t, summary, []*shardSummary{
//		{ShardNum: 0, SeqData: SequenceData{CurrentOrdinal: 5, LastGlobalSeq: 7}},
//		{ShardNum: 1, SeqData: SequenceData{CurrentOrdinal: 2, LastGlobalSeq: 2}},
//	})
//
//}
