/*
 * Copyright (C) 2018 The ontology Authors
 * This file is part of The ontology library.
 *
 * The ontology is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The ontology is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The ontology.  If not, see <http://www.gnu.org/licenses/>.
 */

package utils

import (
	"fmt"
	"io"
	"math/big"

	"github.com/ontio/multi-chain/common"
	"github.com/ontio/multi-chain/common/serialization"
	"github.com/ontio/multi-chain/vm/neovm/types"
)

func WriteVarUint(w io.Writer, value uint64) error {
	if err := serialization.WriteVarBytes(w, types.BigIntToBytes(big.NewInt(int64(value)))); err != nil {
		return fmt.Errorf("serialize value error:%v", err)
	}
	return nil
}

func ReadVarUint(r io.Reader) (uint64, error) {
	value, err := serialization.ReadVarBytes(r)
	if err != nil {
		return 0, fmt.Errorf("deserialize value error:%v", err)
	}
	v := types.BigIntFromBytes(value)
	if v.Cmp(big.NewInt(0)) < 0 {
		return 0, fmt.Errorf("%s", "value should not be a negative number.")
	}
	return v.Uint64(), nil
}

func WriteAddress(w io.Writer, address common.Address) error {
	if err := serialization.WriteVarBytes(w, address[:]); err != nil {
		return fmt.Errorf("serialize value error:%v", err)
	}
	return nil
}

func ReadAddress(r io.Reader) (common.Address, error) {
	from, err := serialization.ReadVarBytes(r)
	if err != nil {
		return common.Address{}, fmt.Errorf("[State] deserialize from error:%v", err)
	}
	return common.AddressParseFromBytes(from)
}

func EncodeAddress(sink *common.ZeroCopySink, addr common.Address) (size uint64) {
	return sink.WriteVarBytes(addr[:])
}

func DecodeAddress(source *common.ZeroCopySource) (common.Address, error) {
	from, _, irregular, eof := source.NextVarBytes()
	if eof {
		return common.Address{}, io.ErrUnexpectedEOF
	}
	if irregular {
		return common.Address{}, common.ErrIrregularData
	}

	return common.AddressParseFromBytes(from)
}

func EncodeVarBytes(sink *common.ZeroCopySink, v []byte) (size uint64) {
	return sink.WriteVarBytes(v)
}

func DecodeVarBytes(source *common.ZeroCopySource) ([]byte, error) {
	v, _, irregular, eof := source.NextVarBytes()
	if eof {
		return nil, io.ErrUnexpectedEOF
	}
	if irregular {
		return nil, common.ErrIrregularData
	}

	return v, nil
}

func EncodeString(sink *common.ZeroCopySink, str string) (size uint64) {
	return sink.WriteVarBytes([]byte(str))
}

func DecodeString(source *common.ZeroCopySource) (string, error) {
	str, _, irregular, eof := source.NextString()
	if eof {
		return "", io.ErrUnexpectedEOF
	}
	if irregular {
		return "", common.ErrIrregularData
	}

	return str, nil
}

func EncodeUint256(sink *common.ZeroCopySink, hash common.Uint256) (size uint64) {
	return sink.WriteVarBytes(hash[:])
}

func DecodeUint256(source *common.ZeroCopySource) (common.Uint256, error) {
	from, _, irregular, eof := source.NextVarBytes()
	if eof {
		return common.Uint256{}, io.ErrUnexpectedEOF
	}
	if irregular {
		return common.Uint256{}, common.ErrIrregularData
	}

	return common.Uint256ParseFromBytes(from)
}

func EncodeVarUint(sink *common.ZeroCopySink, value uint64) (size uint64) {
	return sink.WriteVarBytes(types.BigIntToBytes(big.NewInt(int64(value))))
}

func DecodeVarUint(source *common.ZeroCopySource) (uint64, error) {
	value, _, irregular, eof := source.NextVarBytes()
	if eof {
		return 0, io.ErrUnexpectedEOF
	}
	if irregular {
		return 0, common.ErrIrregularData
	}
	v := types.BigIntFromBytes(value)
	if v.Cmp(big.NewInt(0)) < 0 {
		return 0, fmt.Errorf("%s", "value should not be a negative number.")
	}
	return v.Uint64(), nil
}

func NeoVmSerializeInteger(buf io.Writer, item *big.Int) error {
	if err := serialization.WriteByte(buf, types.IntegerType); err != nil {
		return fmt.Errorf("NeoVmSerializeInteger error: %s", err)
	}
	if err := serialization.WriteVarBytes(buf, common.BigIntToNeoBytes(item)); err != nil {
		return fmt.Errorf("NeoVmSerializeInteger error: %s", err)
	}
	return nil
}

func NeoVmSerializeByteArray(buf io.Writer, item []byte) error {
	if err := serialization.WriteByte(buf, types.ByteArrayType); err != nil {
		return fmt.Errorf("NeoVmSerializeInteger error: %s", err)
	}
	if err := serialization.WriteVarBytes(buf, item); err != nil {
		return fmt.Errorf("NeoVmSerializeInteger error: %s", err)
	}
	return nil
}

func NeoVmSerializeAddress(buf io.Writer, item common.Address) error {
	return NeoVmSerializeByteArray(buf, item[:])
}
