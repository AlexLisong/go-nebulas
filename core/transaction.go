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
	"errors"
	"fmt"
	"time"

	"github.com/nebulasio/go-nebulas/crypto/sha3"

	"github.com/nebulasio/go-nebulas/core/state"

	"encoding/json"

	"github.com/gogo/protobuf/proto"
	"github.com/nebulasio/go-nebulas/core/pb"
	"github.com/nebulasio/go-nebulas/crypto/hash"
	"github.com/nebulasio/go-nebulas/crypto/keystore"
	"github.com/nebulasio/go-nebulas/util"
	"github.com/nebulasio/go-nebulas/util/byteutils"
	"github.com/nebulasio/go-nebulas/util/logging"
	"github.com/sirupsen/logrus"
)

const (
	// TxHashByteLength invalid tx hash length(len of []byte)
	TxHashByteLength = 32
)

var (
	// TransactionMaxGasPrice max gasPrice:50 * 10 ** 9
	TransactionMaxGasPrice, _ = util.NewUint128FromString("50000000000")

	// TransactionMaxGas max gas:50 * 10 ** 9
	TransactionMaxGas, _ = util.NewUint128FromString("50000000000")

	// TransactionGasPrice default gasPrice : 10**6
	TransactionGasPrice, _ = util.NewUint128FromInt(1000000)

	// MinGasCountPerTransaction default gas for normal transaction
	MinGasCountPerTransaction, _ = util.NewUint128FromInt(20000) // TODO check value. use nas = 1000 RMB by default.

	// GasCountPerByte per byte of data attached to a transaction gas cost
	GasCountPerByte, _ = util.NewUint128FromInt(1)

	// MaxDataPayLoadLength Max data length in transaction
	MaxDataPayLoadLength = 128 * 1024
)

// TransactionEvent transaction event
type TransactionEvent struct {
	Hash    string `json:"hash"`
	Status  int8   `json:"status"`
	GasUsed string `json:"gas_used"`
	Error   string `json:"error"`
}

// Transaction type is used to handle all transaction data.
type Transaction struct {
	hash      byteutils.Hash
	from      *Address
	to        *Address
	value     *util.Uint128
	nonce     uint64
	timestamp int64
	data      *corepb.Data
	chainID   uint32
	gasPrice  *util.Uint128
	gasLimit  *util.Uint128

	// Signature
	alg  keystore.Algorithm
	sign byteutils.Hash // Signature values
}

// From return from address
func (tx *Transaction) From() *Address {
	return tx.from
}

// Timestamp return timestamp
func (tx *Transaction) Timestamp() int64 {
	return tx.timestamp
}

// To return to address
func (tx *Transaction) To() *Address {
	return tx.to
}

// ChainID return chainID
func (tx *Transaction) ChainID() uint32 {
	return tx.chainID
}

// Value return tx value
func (tx *Transaction) Value() *util.Uint128 {
	return tx.value
}

// Nonce return tx nonce
func (tx *Transaction) Nonce() uint64 {
	return tx.nonce
}

// Type return tx type
func (tx *Transaction) Type() string {
	return tx.data.Type
}

// Data return tx data
func (tx *Transaction) Data() []byte {
	return tx.data.Payload
}

// ToProto converts domain Tx to proto Tx
func (tx *Transaction) ToProto() (proto.Message, error) {
	value, err := tx.value.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	gasPrice, err := tx.gasPrice.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	gasLimit, err := tx.gasLimit.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	return &corepb.Transaction{
		Hash:      tx.hash,
		From:      tx.from.address,
		To:        tx.to.address,
		Value:     value,
		Nonce:     tx.nonce,
		Timestamp: tx.timestamp,
		Data:      tx.data,
		ChainId:   tx.chainID,
		GasPrice:  gasPrice,
		GasLimit:  gasLimit,
		Alg:       uint32(tx.alg),
		Sign:      tx.sign,
	}, nil
}

// FromProto converts proto Tx into domain Tx
func (tx *Transaction) FromProto(msg proto.Message) error {
	if msg, ok := msg.(*corepb.Transaction); ok {
		if msg != nil {
			tx.hash = msg.Hash
			from, err := AddressParseFromBytes(msg.From)
			if err != nil {
				return err
			}
			tx.from = from

			to, err := AddressParseFromBytes(msg.To)
			if err != nil {
				return err
			}
			tx.to = to

			value, err := util.NewUint128FromFixedSizeByteSlice(msg.Value)
			if err != nil {
				return err
			}
			tx.value = value

			tx.nonce = msg.Nonce
			tx.timestamp = msg.Timestamp
			tx.chainID = msg.ChainId

			if msg.Data == nil {
				return ErrInvalidTransactionData
			}
			if len(msg.Data.Payload) > MaxDataPayLoadLength {
				return ErrTxDataPayLoadOutOfMaxLength
			}
			tx.data = msg.Data

			gasPrice, err := util.NewUint128FromFixedSizeByteSlice(msg.GasPrice)
			if err != nil {
				return err
			}
			tx.gasPrice = gasPrice

			gasLimit, err := util.NewUint128FromFixedSizeByteSlice(msg.GasLimit)
			if err != nil {
				return err
			}
			tx.gasLimit = gasLimit

			tx.alg = keystore.Algorithm(msg.Alg)
			tx.sign = msg.Sign
			return nil
		}
		return ErrInvalidProtoToTransaction
	}
	return ErrInvalidProtoToTransaction
}

func (tx *Transaction) String() string {
	return fmt.Sprintf(`{"chainID":%d, "hash":"%s", "from":"%s", "to":"%s", "nonce":%d, "value":"%s", "timestamp":%d, "gasprice": "%s", "gaslimit":"%s", "type":"%s"}`,
		tx.chainID,
		tx.hash.String(),
		tx.from.String(),
		tx.to.String(),
		tx.nonce,
		tx.value.String(),
		tx.timestamp,
		tx.gasPrice.String(),
		tx.gasLimit.String(),
		tx.Type(),
	)
}

// Transactions is an alias of Transaction array.
type Transactions []*Transaction

// NewTransaction create #Transaction instance.
func NewTransaction(chainID uint32, from, to *Address, value *util.Uint128, nonce uint64, payloadType string, payload []byte, gasPrice *util.Uint128, gasLimit *util.Uint128) (*Transaction, error) {
	if gasPrice == nil || gasPrice.Cmp(util.NewUint128()) <= 0 {
		return nil, ErrZeroGasPrice
	}
	if gasLimit == nil || gasLimit.Cmp(util.NewUint128()) <= 0 {
		return nil, ErrZeroGasLimit
	}

	if nil == from || nil == to || nil == value {
		logging.VLog().WithFields(logrus.Fields{
			"from":  from,
			"to":    to,
			"value": value,
		}).Error("invalid parameters")
		return nil, ErrInvalidArgument
	}

	if len(payload) > MaxDataPayLoadLength {
		return nil, ErrTxDataPayLoadOutOfMaxLength
	}

	tx := &Transaction{
		from:      from,
		to:        to,
		value:     value,
		nonce:     nonce,
		timestamp: time.Now().Unix(),
		chainID:   chainID,
		data:      &corepb.Data{Type: payloadType, Payload: payload},
		gasPrice:  gasPrice,
		gasLimit:  gasLimit,
	}
	return tx, nil
}

// Hash return the hash of transaction.
func (tx *Transaction) Hash() byteutils.Hash {
	return tx.hash
}

// GasPrice returns gasPrice
func (tx *Transaction) GasPrice() *util.Uint128 {
	return tx.gasPrice
}

// GasLimit returns gasLimit
func (tx *Transaction) GasLimit() *util.Uint128 {
	return tx.gasLimit
}

// PayloadGasLimit returns payload gasLimit
func (tx *Transaction) PayloadGasLimit(payload TxPayload) (*util.Uint128, error) { // TODO delete
	if payload == nil {
		return nil, ErrNilArgument
	}

	// TODO: @robin using CalculateMinGasExpected instead.
	// payloadGasLimit = tx.gasLimit - tx.GasCountOfTxBase - payload.BaseGasCount

	gasCountOfTxBase, err := tx.GasCountOfTxBase()
	if err != nil {
		return nil, err
	}
	payloadGasLimit, err := tx.gasLimit.Sub(gasCountOfTxBase)
	if err != nil {
		return nil, ErrOutOfGasLimit
	}
	payloadGasLimit, err = payloadGasLimit.Sub(payload.BaseGasCount())
	if err != nil {
		return nil, ErrOutOfGasLimit
	}
	return payloadGasLimit, nil
}

// MinBalanceRequired returns gasprice * gaslimit + tx.value.
func (tx *Transaction) MinBalanceRequired() (*util.Uint128, error) { // TODO = gasPrice * gasLimit
	total, err := tx.GasPrice().Mul(tx.GasLimit())
	if err != nil {
		return nil, err
	}
	total, err = total.Add(tx.value)
	if err != nil {
		return nil, err
	}
	return total, nil
}

// GasCountOfTxBase calculate the actual amount for a tx with data
func (tx *Transaction) GasCountOfTxBase() (*util.Uint128, error) {
	txGas := MinGasCountPerTransaction
	if tx.DataLen() > 0 {
		dataLen, err := util.NewUint128FromInt(int64(tx.DataLen()))
		if err != nil {
			return nil, err
		}
		dataGas, err := dataLen.Mul(GasCountPerByte)
		if err != nil {
			return nil, err
		}
		txGas, err = txGas.Add(dataGas)
		if err != nil {
			return nil, err
		}
	}
	return txGas, nil
}

// DataLen return the length of payload
func (tx *Transaction) DataLen() int {
	return len(tx.data.Payload)
}

// LoadPayload returns tx's payload
func (tx *Transaction) LoadPayload() (TxPayload, error) {
	// execute payload
	var (
		payload TxPayload
		err     error
	)
	switch tx.data.Type {
	case TxPayloadBinaryType:
		payload, err = LoadBinaryPayload(tx.data.Payload)
	case TxPayloadDeployType:
		payload, err = LoadDeployPayload(tx.data.Payload)
	case TxPayloadCallType:
		payload, err = LoadCallPayload(tx.data.Payload)
	default:
		err = ErrInvalidTxPayloadType
	}
	return payload, err
}

// CalculateMinGasExpected calculate min gas expected for a transaction to put on chain.
// MinGasExpected = GasCountOfBase + payload.baseGasCount
func (tx *Transaction) CalculateMinGasExpected(payload TxPayload) (*util.Uint128, TxPayload, error) {
	gasUsed, err := tx.GasCountOfTxBase()
	if err != nil {
		return nil, payload, err
	}

	// load payload if nil.
	if payload == nil {
		payload, err = tx.LoadPayload()
		if err != nil {
			return nil, nil, err
		}
	}

	gasUsed, err = gasUsed.Add(payload.BaseGasCount())
	if err != nil {
		return nil, payload, err
	}

	return gasUsed, payload, nil
}

// VerifyExecution transaction and return result.
func VerifyExecution(tx *Transaction, block *Block, ws WorldState) (bool, error) {
	// step1. check balance >= gasLimit * gasPrice + value // TODO check balance >= gasLimit * gasPrice
	if giveback, err := tx.checkBalance(block, ws); err != nil {
		return giveback, err
	}

	// TODO: @robin using CalculateMinGasExpected instead.

	// step2. calculate base gas
	gasUsed, err := tx.GasCountOfTxBase()
	if err != nil {
		logging.VLog().Info("VEE 1")
		// Gas overflow, won't giveback the tx
		return false, err
	}
	if tx.gasLimit.Cmp(gasUsed) < 0 {
		logging.VLog().WithFields(logrus.Fields{
			"error":       ErrOutOfGasLimit,
			"transaction": tx,
			"limit":       tx.gasLimit,
			"used":        gasUsed,
		}).Debug("Failed to check tx based gas used.")
		// GasLimit is smaller than based tx gas, won't giveback the tx
		return false, ErrOutOfGasLimit
	}

	// step3. check payload vaild. all txs come here can be submitted on chain.
	payload, payloadErr := tx.LoadPayload()
	if payloadErr != nil {
		logging.VLog().WithFields(logrus.Fields{
			"payloadErr":  payloadErr,
			"block":       block,
			"transaction": tx,
		}).Debug("Failed to load payload.")
		go metricsTxExeFailed.Mark(1)

		if err := tx.recordGas(gasUsed, ws); err != nil {
			logging.VLog().Info("AEE 2")
			// Gas overflow, won't giveback the tx
			return false, err
		}
		if err := tx.recordResultEvent(gasUsed, payloadErr, ws); err != nil {
			return true, err
		}
		// No error, won't giveback the tx
		return false, nil
	}

	// step4. check gasLimit > gas + payload.baseGasCount
	gasUsed, err = gasUsed.Add(payload.BaseGasCount())
	if err != nil {
		logging.VLog().Info("AEE 3")
		// Gas overflow, won't giveback the tx
		return false, err
	}
	if tx.gasLimit.Cmp(gasUsed) < 0 {
		logging.VLog().WithFields(logrus.Fields{
			"err":   ErrOutOfGasLimit,
			"block": block,
			"tx":    tx,
		}).Debug("Failed to check base gas used.")
		go metricsTxExeFailed.Mark(1)

		if err := tx.recordGas(tx.gasLimit, ws); err != nil {
			logging.VLog().Info("AEE 4")
			// Gas overflow, won't giveback the tx
			return false, err
		}
		if err := tx.recordResultEvent(tx.gasLimit, ErrOutOfGasLimit, ws); err != nil {
			return true, err
		}
		// No error, won't giveback the tx
		return false, nil
	}

	// step5. transfer value.
	if giveback, err := transfer(tx.from.address, tx.to.address, tx.value, ws); err != nil { // TODO check balance sufficient. balance >= gasLimit * gasPrice + value
		return giveback, err
	}

	// step6. execute payload
	// execute smart contract and sub the calcute gas.
	gasExecution, _, exeErr := payload.Execute(tx, block, ws) // TODO return detailed error, not only failed // TODO calculate the payload gaslimit as an argument in Execute()
	if exeErr != nil {
		logging.VLog().Info("Reset Payload ", tx, " err ", exeErr)
		if err := ws.Reset(); err != nil {
			logging.VLog().Info("AEE 5")
			return true, err
		}
	}

	allGas, gasErr := gasUsed.Add(gasExecution)
	if gasErr != nil {
		logging.VLog().Info("AEE 6")
		// Gas overflow, won't giveback the tx
		return false, gasErr
	}
	if tx.gasLimit.Cmp(allGas) < 0 {
		logging.VLog().WithFields(logrus.Fields{
			"err":   ErrOutOfGasLimit,
			"block": block,
			"tx":    tx,
		}).Debug("Failed to check gas executed.")
		go metricsTxExeFailed.Mark(1)

		if err := ws.Reset(); err != nil {
			logging.VLog().Info("AEE 7")
			return true, err
		}
		if err := tx.recordGas(tx.gasLimit, ws); err != nil {
			logging.VLog().Info("AEE 8")
			// Gas overflow, won't giveback the tx
			return false, err
		}
		if err := tx.recordResultEvent(tx.gasLimit, ErrOutOfGasLimit, ws); err != nil {
			return true, err
		}
		// No error, won't giveback the tx
		return false, nil
	}

	if err := tx.recordGas(allGas, ws); err != nil {
		// Gas overflow, won't giveback the tx
		return false, err
	}
	if err := tx.recordResultEvent(allGas, exeErr, ws); err != nil {
		return true, err
	}

	if exeErr != nil {
		logging.VLog().WithFields(logrus.Fields{
			"exeErr":       exeErr,
			"block":        block,
			"tx":           tx,
			"gasUsed":      gasUsed,
			"gasExecution": gasExecution,
		}).Debug("Failed to execute payload.")
		go metricsTxExeFailed.Mark(1)
	} else {
		go metricsTxExeSuccess.Mark(1)
	}
	// No error, won't giveback the tx
	return false, nil
}

// SimulateExecution simulate execution and return gasUsed, executionResult and error if occurred.
func (tx *Transaction) SimulateExecution(block *Block) (*util.Uint128, string, error, error) {
	// prepare gasLimit to TransactionMaxGas.
	tx.gasLimit = TransactionMaxGas

	// hash is necessary in nvm
	hash, err := tx.calHash()
	if err != nil {
		return nil, "", nil, err
	}
	tx.hash = hash

	// calculate min gas.
	gasUsed, payload, err := tx.CalculateMinGasExpected(nil)
	if err != nil {
		return nil, "", nil, err
	}

	var (
		result string
		exeErr error
	)

	ws := block.WorldState()

	// try run smart contract if payload is.
	if tx.data.Type == TxPayloadCallType || tx.data.Type == TxPayloadDeployType {

		// transfer value to smart contract.
		toAcc, err := ws.GetOrCreateUserAccount(tx.to.address)
		if err != nil {
			return nil, "", nil, err
		}
		err = toAcc.AddBalance(tx.value)
		if err != nil {
			return nil, "", nil, err
		}

		// execute.
		var gasExecution *util.Uint128
		gasExecution, result, exeErr = payload.Execute(tx, block, block.WorldState())

		// add gas.
		gasUsed, err = gasUsed.Add(gasExecution)
		if err != nil {
			return nil, "", nil, err
		}

		if exeErr != nil {
			return gasUsed, result, exeErr, nil
		}
	}

	// check balance.
	ok, err := checkBalanceForGasUsedAndValue(ws, tx.from.address, tx.value, gasUsed, tx.gasPrice)
	if err == nil && ok == false {
		exeErr = ErrInsufficientBalance
	}

	return gasUsed, result, exeErr, err
}

func (tx *Transaction) checkBalance(block *Block, ws WorldState) (bool, error) {
	// TODO: @robin using checkBalanceForGasLimit instead.
	fromAcc, err := ws.GetOrCreateUserAccount(tx.from.address)
	if err != nil {
		logging.VLog().Info("AEE 10")
		return true, err
	}
	minBalanceRequired, err := tx.MinBalanceRequired()
	if err != nil {
		logging.VLog().Info("AEE 11")
		// MinBalanceRequired is not uint128, won't giveback the tx
		return false, err
	}
	if fromAcc.Balance().Cmp(minBalanceRequired) < 0 {
		// Balance is smaller than min balance required, won't giveback the tx
		return false, ErrInsufficientBalance
	}
	// No error, won't giveback the tx
	return false, nil
}

// checkBalanceForGasLimit check balance >= gasLimit * gasPrice.
func (tx *Transaction) checkBalanceForGasLimit(block *Block, ws WorldState) error {
	ok, err := checkBalanceForGasUsedAndValue(ws, tx.from.address, util.NewUint128(), tx.gasLimit, tx.gasPrice)
	if err != nil {
		return err
	}

	if ok == false {
		return ErrInsufficientBalance
	}

	return nil
}

// checkBalanceForGasUsedAndValue check balance >= gasUsed * gasPrice + value.
func checkBalanceForGasUsedAndValue(ws WorldState, address byteutils.Hash, value, gasUsed, gasPrice *util.Uint128) (bool, error) {
	fromAcc, err := ws.GetOrCreateUserAccount(address)
	if err != nil {
		return false, err
	}
	gasFee, err := gasPrice.Mul(gasUsed)
	if err != nil {
		return false, err
	}
	balanceRequired, err := gasFee.Add(value)
	if err != nil {
		return false, err
	}
	if fromAcc.Balance().Cmp(balanceRequired) < 0 {
		return false, nil
	}
	return true, nil

}

func (tx *Transaction) recordGas(gasCnt *util.Uint128, ws WorldState) error {
	gasCost, err := tx.GasPrice().Mul(gasCnt)
	if err != nil {
		logging.VLog().Info("AEE 12")
		return err
	}

	return ws.RecordGas(tx.from.String(), gasCost)
}

func transfer(from, to byteutils.Hash, value *util.Uint128, ws WorldState) (bool, error) {
	fromAcc, err := ws.GetOrCreateUserAccount(from)
	if err != nil {
		logging.VLog().Info("AEE 13")
		return true, err
	}
	toAcc, err := ws.GetOrCreateUserAccount(to)
	if err != nil {
		logging.VLog().Info("AEE 14")
		return true, err
	}
	if err := fromAcc.SubBalance(value); err != nil {
		logging.VLog().Info("AEE 15")
		// Balance is not enough to transfer the value, won't giveback the tx
		return false, err
	}
	if err := toAcc.AddBalance(value); err != nil {
		logging.VLog().Info("AEE 16")
		// Balance plus value result in overflow, won't giveback the tx
		return false, err
	}
	// No error, won't giveback the tx
	return false, nil
}

func (tx *Transaction) recordResultEvent(gasUsed *util.Uint128, err error, ws WorldState) error {
	txEvent := &TransactionEvent{
		Hash:    tx.hash.String(),
		GasUsed: gasUsed.String(),
	}
	if err != nil {
		txEvent.Status = TxExecutionFailed
		txEvent.Error = err.Error() // TODO limit the length of err
	} else {
		txEvent.Status = TxExecutionSuccess
	}

	txData, err := json.Marshal(txEvent)
	if err != nil {
		return err
	}

	event := &state.Event{
		Topic: TopicTransactionExecutionResult,
		Data:  string(txData),
	}
	return ws.RecordEvent(tx.hash, event)
}

// Sign sign transaction,sign algorithm is
func (tx *Transaction) Sign(signature keystore.Signature) error {
	if signature == nil {
		return ErrNilArgument
	}
	hash, err := tx.calHash()
	if err != nil {
		return err
	}
	sign, err := signature.Sign(hash)
	if err != nil {
		return err
	}
	tx.hash = hash
	tx.alg = signature.Algorithm()
	tx.sign = sign
	return nil
}

// VerifyIntegrity return transaction verify result, including Hash and Signature.
func (tx *Transaction) VerifyIntegrity(chainID uint32) error {
	// check ChainID.
	if tx.chainID != chainID {
		return ErrInvalidChainID
	}

	// check Hash.
	wantedHash, err := tx.calHash()
	if err != nil {
		return err
	}
	if wantedHash.Equals(tx.hash) == false {
		return ErrInvalidTransactionHash
	}

	// check Signature.
	return tx.verifySign()

}

func (tx *Transaction) verifySign() error { // TODO move to core/crypto.go.
	signer, err := RecoverSignerFromSignature(tx.alg, tx.hash, tx.sign)
	if err != nil {
		return err
	}
	if !tx.from.Equals(signer) {
		logging.VLog().WithFields(logrus.Fields{
			"signer":  signer.String(),
			"tx.from": tx.from,
		}).Debug("Failed to verify tx's sign.")
		return ErrInvalidTransactionSigner
	}
	return nil
}

// GenerateContractAddress according to tx.from and tx.nonce.
func (tx *Transaction) GenerateContractAddress() (*Address, error) {
	if TxPayloadDeployType != tx.Type() {
		return nil, errors.New("playload type err")
	}
	return NewContractAddressFromHash(hash.Sha3256(tx.from.Bytes(), byteutils.FromUint64(tx.nonce)))
}

// CheckContract check if contract is valid
func CheckContract(addr *Address, ws WorldState) (state.Account, error) {
	contract, err := ws.GetContractAccount(addr.Bytes())
	if err != nil {
		return nil, err
	}

	if len(contract.BirthPlace()) == 0 { // TODO check nil, move if-else into ws.GetContractAccount
		return nil, ErrContractCheckFailed
	}

	birthEvents, err := ws.FetchEvents(contract.BirthPlace())
	if err != nil {
		return nil, err
	}

	result := false
	for _, v := range birthEvents {

		if v.Topic == TopicTransactionExecutionResult { // TODO use the last event
			txEvent := TransactionEvent{}
			if err := json.Unmarshal([]byte(v.Data), &txEvent); err != nil {
				return nil, err
			}
			if txEvent.Status == TxExecutionSuccess {
				result = true
				break
			}
		}
	}
	if !result {
		return nil, ErrContractCheckFailed
	}

	return contract, nil
}

// CheckTransaction in a tx world state
func CheckTransaction(tx *Transaction, ws WorldState) (bool, error) {
	// check nonce
	fromAcc, err := ws.GetOrCreateUserAccount(tx.from.address)
	if err != nil {
		logging.VLog().Info("CTE 1")
		return true, err
	}

	// pass current Nonce.
	currentNonce := fromAcc.Nonce()

	if tx.nonce < currentNonce+1 {
		// Nonce is too small, won't giveback the tx
		return false, ErrSmallTransactionNonce
	} else if tx.nonce > currentNonce+1 {
		return true, ErrLargeTransactionNonce
	}

	return false, nil
}

// AcceptTransaction in a tx world state
func AcceptTransaction(tx *Transaction, ws WorldState) (bool, error) {
	// record tx
	pbTx, err := tx.ToProto()
	if err != nil {
		logging.VLog().Info("ATE 1")
		return true, err
	}
	txBytes, err := proto.Marshal(pbTx)
	if err != nil {
		logging.VLog().Info("ATE 2")
		return true, err
	}
	if err := ws.PutTx(tx.hash, txBytes); err != nil {
		logging.VLog().Info("ATE 3")
		return true, err
	}
	// incre nonce
	fromAcc, err := ws.GetOrCreateUserAccount(tx.from.address)
	if err != nil {
		logging.VLog().Info("ATE 4")
		return true, err
	}
	fromAcc.IncrNonce()
	// No error, won't giveback the tx
	return false, nil
}

// GetTransaction from txs Trie
func GetTransaction(hash byteutils.Hash, ws WorldState) (*Transaction, error) {
	if len(hash) != TxHashByteLength {
		return nil, ErrInvalidArgument
	}
	bytes, err := ws.GetTx(hash)
	if err != nil {
		return nil, err
	}
	pbTx := new(corepb.Transaction)
	if err := proto.Unmarshal(bytes, pbTx); err != nil {
		return nil, err
	}
	tx := new(Transaction)
	if err = tx.FromProto(pbTx); err != nil {
		return nil, err
	}
	return tx, nil
}

// HashTransaction hash the transaction.
func (tx *Transaction) calHash() (byteutils.Hash, error) {
	hasher := sha3.New256()

	value, err := tx.value.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	data, err := proto.Marshal(tx.data)
	if err != nil {
		return nil, err
	}
	gasPrice, err := tx.gasPrice.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}
	gasLimit, err := tx.gasLimit.ToFixedSizeByteSlice()
	if err != nil {
		return nil, err
	}

	hasher.Write(tx.from.address)
	hasher.Write(tx.to.address)
	hasher.Write(value)
	hasher.Write(byteutils.FromUint64(tx.nonce))
	hasher.Write(byteutils.FromInt64(tx.timestamp))
	hasher.Write(data)
	hasher.Write(byteutils.FromUint32(tx.chainID))
	hasher.Write(gasPrice)
	hasher.Write(gasLimit)

	return hasher.Sum(nil), nil
}
