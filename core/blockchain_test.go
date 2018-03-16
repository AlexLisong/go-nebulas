// Copyright (C) 2017 go-nebulas authors
//
// This file is part of the go-nebulas library.
//
// the go-nebulas library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-nebulas library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-nebulas library.  If not, see <http://www.gnu.org/licenses/>.
//

package core

import (
	"testing"
	"time"

	"github.com/nebulasio/go-nebulas/crypto"
	"github.com/nebulasio/go-nebulas/crypto/keystore"
	"github.com/nebulasio/go-nebulas/crypto/keystore/secp256k1"
	"github.com/nebulasio/go-nebulas/util"

	"github.com/stretchr/testify/assert"
)

func TestBlockChain_FindCommonAncestorWithTail(t *testing.T) {
	neb := testNeb(t)
	bc := neb.chain

	ks := keystore.DefaultKS
	priv := secp256k1.GeneratePrivateKey()
	pubdata, _ := priv.PublicKey().Encoded()
	from, _ := NewAddressFromPublicKey(pubdata)
	ks.SetKey(from.String(), priv, []byte("passphrase"))
	ks.Unlock(from.String(), []byte("passphrase"), time.Second*60*60*24*365)

	key, _ := ks.GetUnlocked(from.String())
	signature, _ := crypto.NewSignature(keystore.SECP256K1)
	signature.InitSign(key.(keystore.PrivateKey))

	//add from reward
	block0, _ := bc.NewBlock(from)
	block0.header.timestamp = BlockInterval
	block0.Seal()
	assert.Nil(t, bc.BlockPool().Push(block0))
	bc.SetTailBlock(block0)
	assert.Equal(t, bc.latestIrreversibleBlock, bc.genesisBlock)

	coinbase11 := mockAddress()
	coinbase12 := mockAddress()
	coinbase111 := mockAddress()
	coinbase221 := mockAddress()
	coinbase222 := mockAddress()
	coinbase1111 := mockAddress()
	coinbase11111 := mockAddress()
	/*
		genesis -- 0 -- 11 -- 111 -- 1111
					 \_ 12 -- 221
					       \_ 222 tail
	*/
	block11, err := bc.NewBlock(coinbase11)
	assert.Nil(t, err)
	block11.header.timestamp = BlockInterval * 2
	block12, err := bc.NewBlock(coinbase12)
	assert.Nil(t, err)
	block12.header.timestamp = BlockInterval * 3
	block11.Seal()
	block12.Seal()
	netBlock11, err := mockBlockFromNetwork(block11)
	assert.Nil(t, err)
	assert.Nil(t, bc.BlockPool().Push(netBlock11))
	netBlock12, err := mockBlockFromNetwork(block12)
	assert.Nil(t, err)
	assert.Nil(t, bc.BlockPool().Push(netBlock12))
	bc.SetTailBlock(block12)
	assert.Equal(t, bc.latestIrreversibleBlock, bc.genesisBlock)
	bc.SetTailBlock(block11)
	assert.Equal(t, bc.latestIrreversibleBlock, bc.genesisBlock)
	block111, _ := bc.NewBlock(coinbase111)
	block111.header.timestamp = BlockInterval * 4
	block111.Seal()
	netBlock111, err := mockBlockFromNetwork(block111)
	assert.Nil(t, err)
	assert.Nil(t, bc.BlockPool().Push(netBlock111))
	bc.SetTailBlock(block12)
	assert.Equal(t, bc.latestIrreversibleBlock, bc.genesisBlock)
	block221, _ := bc.NewBlock(coinbase221)
	block221.header.timestamp = BlockInterval * 5
	block222, _ := bc.NewBlock(coinbase222)
	block222.header.timestamp = BlockInterval * 6
	block221.Seal()
	block222.Seal()
	netBlock221, err := mockBlockFromNetwork(block221)
	assert.Nil(t, err)
	assert.Nil(t, bc.BlockPool().Push(netBlock221))
	netBlock222, err := mockBlockFromNetwork(block222)
	assert.Nil(t, err)
	assert.Nil(t, bc.BlockPool().Push(netBlock222))
	bc.SetTailBlock(block111)
	assert.Equal(t, bc.latestIrreversibleBlock, bc.genesisBlock)
	block1111, _ := bc.NewBlock(coinbase1111)
	block1111.header.timestamp = BlockInterval * 7
	block1111.Seal()
	netBlock1111, err := mockBlockFromNetwork(block1111)
	assert.Nil(t, err)
	assert.Nil(t, bc.BlockPool().Push(netBlock1111))
	bc.SetTailBlock(block222)
	assert.Equal(t, bc.latestIrreversibleBlock, bc.genesisBlock)
	tails := bc.DetachedTailBlocks()
	for _, v := range tails {
		if v.Hash().Equals(block221.Hash()) ||
			v.Hash().Equals(block222.Hash()) ||
			v.Hash().Equals(block1111.Hash()) {
			continue
		}
		assert.Equal(t, true, false)
	}
	assert.Equal(t, len(tails), 3)

	netBlock1111, err = mockBlockFromNetwork(block1111)
	assert.Nil(t, err)
	common1, err := bc.FindCommonAncestorWithTail(netBlock1111)
	assert.Nil(t, err)

	netBlock0, err := mockBlockFromNetwork(block0)
	assert.Nil(t, err)
	netCommon1, err := mockBlockFromNetwork(common1)
	assert.Nil(t, err)
	assert.Equal(t, netCommon1, netBlock0)

	netBlock221, err = mockBlockFromNetwork(block221)
	assert.Nil(t, err)
	common2, err := bc.FindCommonAncestorWithTail(netBlock221)
	assert.Nil(t, err)
	netCommon2, err := mockBlockFromNetwork(common2)
	assert.Nil(t, err)
	netBlock12, err = mockBlockFromNetwork(block12)
	assert.Nil(t, err)
	assert.Equal(t, netCommon2, netBlock12)

	netBlock222, err = mockBlockFromNetwork(block222)
	assert.Nil(t, err)
	common3, err := bc.FindCommonAncestorWithTail(netBlock222)
	assert.Nil(t, err)
	netCommon3, err := mockBlockFromNetwork(common3)
	assert.Nil(t, err)
	assert.Equal(t, netCommon3, netBlock222)

	netTail, err := mockBlockFromNetwork(bc.tailBlock)
	assert.Nil(t, err)
	common4, err := bc.FindCommonAncestorWithTail(netTail)
	assert.Nil(t, err)
	netCommon4, err := mockBlockFromNetwork(common4)
	assert.Nil(t, err)
	assert.Equal(t, netCommon4, netTail)

	netBlock12, err = mockBlockFromNetwork(block12)
	assert.Nil(t, err)
	common5, err := bc.FindCommonAncestorWithTail(netBlock12)
	assert.Nil(t, err)
	netCommon5, err := mockBlockFromNetwork(common5)
	assert.Nil(t, err)
	assert.Equal(t, netCommon5, netBlock12)

	result := bc.Dump(4)
	assert.Equal(t, result, "["+block222.String()+","+block12.String()+","+block0.String()+","+bc.genesisBlock.String()+"]")

	bc.SetTailBlock(block1111)
	assert.Equal(t, bc.latestIrreversibleBlock, bc.genesisBlock)

	block11111, _ := bc.NewBlock(coinbase11111)
	block11111.header.timestamp = BlockInterval * 8
	block11111.Seal()
	netBlock11111, err := mockBlockFromNetwork(block11111)
	assert.Nil(t, err)
	assert.Nil(t, bc.BlockPool().Push(netBlock11111))
	bc.SetTailBlock(block11111)
}

func TestBlockChain_FetchDescendantInCanonicalChain(t *testing.T) {
	neb := testNeb(t)
	bc := neb.chain

	coinbase := &Address{[]byte("012345678901234567890000")}
	/*
		genesisi -- 1 - 2 - 3 - 4 - 5 - 6
		         \_ block - block1
	*/
	block, _ := bc.NewBlock(coinbase)
	block.header.timestamp = BlockInterval
	block.Seal()
	bc.BlockPool().Push(block)
	bc.SetTailBlock(block)

	block1, _ := bc.NewBlock(coinbase)
	block1.header.timestamp = BlockInterval * 2
	block1.Seal()
	bc.BlockPool().Push(block1)
	bc.SetTailBlock(block1)

	var blocks []*Block
	for i := 0; i < 6; i++ {
		block, _ := bc.NewBlock(coinbase)
		block.header.timestamp = BlockInterval * int64(i+3)
		blocks = append(blocks, block)
		block.Seal()
		bc.BlockPool().Push(block)
		bc.SetTailBlock(block)
	}
	blocks24, _ := bc.FetchDescendantInCanonicalChain(3, blocks[0])
	assert.Equal(t, blocks24[0].Hash(), blocks[1].Hash())
	assert.Equal(t, blocks24[1].Hash(), blocks[2].Hash())
	assert.Equal(t, blocks24[2].Hash(), blocks[3].Hash())
	blocks46, _ := bc.FetchDescendantInCanonicalChain(10, blocks[2])
	assert.Equal(t, len(blocks46), 3)
	assert.Equal(t, blocks46[0].Hash(), blocks[3].Hash())
	assert.Equal(t, blocks46[1].Hash(), blocks[4].Hash())
	assert.Equal(t, blocks46[2].Hash(), blocks[5].Hash())
	blocks13, _ := bc.FetchDescendantInCanonicalChain(3, bc.genesisBlock)
	assert.Equal(t, len(blocks13), 3)
	blocks0, err0 := bc.FetchDescendantInCanonicalChain(3, blocks[5])
	assert.Equal(t, len(blocks0), 0)
	assert.Nil(t, err0)
}

func TestBlockChain_EstimateGas(t *testing.T) {
	priv := secp256k1.GeneratePrivateKey()
	pubdata, _ := priv.PublicKey().Encoded()
	from, _ := NewAddressFromPublicKey(pubdata)
	to := &Address{from.address}

	payload, err := NewBinaryPayload(nil).ToBytes()
	assert.Nil(t, err)

	neb := testNeb(t)
	bc := neb.chain
	gasLimit, _ := util.NewUint128FromInt(200000)
	tx := NewTransaction(bc.ChainID(), from, to, util.NewUint128(), 1, TxPayloadBinaryType, payload, TransactionGasPrice, gasLimit)

	_, err = bc.EstimateGas(tx)
	assert.Nil(t, err)
}

func TestTailBlock(t *testing.T) {
	neb := testNeb(t)
	bc := neb.chain
	block, err := bc.LoadTailFromStorage()
	assert.Nil(t, err)
	assert.Equal(t, bc.tailBlock.Hash(), block.Hash())
}

func TestGetPrice(t *testing.T) {
	neb := testNeb(t)
	bc := neb.chain
	assert.Equal(t, bc.GasPrice(), TransactionGasPrice)

	ks := keystore.DefaultKS
	from := mockAddress()
	key, err := ks.GetUnlocked(from.String())
	assert.Nil(t, err)
	signature, err := crypto.NewSignature(keystore.SECP256K1)
	assert.Nil(t, err)
	signature.InitSign(key.(keystore.PrivateKey))
	block, err := bc.NewBlock(from)
	assert.Nil(t, err)
	GasPriceDetla, _ := util.NewUint128FromInt(1)
	lowerGasPrice, err := TransactionGasPrice.Sub(GasPriceDetla)
	assert.Nil(t, err)
	gasLimit, _ := util.NewUint128FromInt(200000)
	tx1 := NewTransaction(bc.ChainID(), from, from, util.NewUint128(), 1, TxPayloadBinaryType, []byte("nas"), lowerGasPrice, gasLimit)
	tx1.Sign(signature)
	tx2 := NewTransaction(bc.ChainID(), from, from, util.NewUint128(), 2, TxPayloadBinaryType, []byte("nas"), TransactionGasPrice, gasLimit)
	tx2.Sign(signature)
	block.transactions = append(block.transactions, tx1)
	block.transactions = append(block.transactions, tx2)
	block.Seal()
	block.Sign(signature)
	bc.SetTailBlock(block)
	assert.Equal(t, bc.GasPrice(), lowerGasPrice)
}
