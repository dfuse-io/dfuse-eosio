package codec

import "github.com/streamingfast/bstream"

func init() {
	bstream.GetBlockWriterFactory = bstream.BlockWriterFactoryFunc(BlockWriterFactory)
	bstream.GetBlockReaderFactory = bstream.BlockReaderFactoryFunc(BlockReaderFactory)
	bstream.GetBlockDecoder = bstream.BlockDecoderFunc(BlockDecoder)
	bstream.GetProtocolFirstStreamableBlock = 2
	bstream.GetProtocolGenesisBlock = 1
	bstream.GetBlockWriterHeaderLen = 10
}
