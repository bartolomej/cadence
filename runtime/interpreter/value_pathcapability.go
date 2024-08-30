/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package interpreter

import (
	"fmt"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

// TODO: remove once migrated

// Deprecated: PathCapabilityValue
type PathCapabilityValue struct {
	BorrowType StaticType
	Path       PathValue
	address    AddressValue
}

var _ Value = &PathCapabilityValue{}
var _ atree.Storable = &PathCapabilityValue{}
var _ EquatableValue = &PathCapabilityValue{}
var _ MemberAccessibleValue = &PathCapabilityValue{}
var _ CapabilityValue = &PathCapabilityValue{}

func NewUnmeteredPathCapabilityValue(
	borrowType StaticType,
	address AddressValue,
	path PathValue,
) *PathCapabilityValue {
	return &PathCapabilityValue{
		BorrowType: borrowType,
		address:    address,
		Path:       path,
	}
}

func (*PathCapabilityValue) isValue() {}

func (*PathCapabilityValue) isCapabilityValue() {}

func (v *PathCapabilityValue) Accept(_ *Interpreter, _ Visitor, _ LocationRange) {
	panic(errors.NewUnreachableError())
}

func (v *PathCapabilityValue) Walk(_ *Interpreter, walkChild func(Value), _ LocationRange) {
	walkChild(v.address)
	walkChild(v.Path)
}

func (v *PathCapabilityValue) StaticType(inter *Interpreter) StaticType {
	return NewCapabilityStaticType(
		inter,
		v.BorrowType,
	)
}

func (v *PathCapabilityValue) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return false
}
func (v *PathCapabilityValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *PathCapabilityValue) RecursiveString(seenReferences SeenReferences) string {
	borrowType := v.BorrowType
	if borrowType == nil {
		return fmt.Sprintf(
			"Capability(address: %s, path: %s)",
			v.address.RecursiveString(seenReferences),
			v.Path.RecursiveString(seenReferences),
		)
	} else {
		return fmt.Sprintf(
			"Capability<%s>(address: %s, path: %s)",
			borrowType.String(),
			v.address.RecursiveString(seenReferences),
			v.Path.RecursiveString(seenReferences),
		)
	}
}

func (v *PathCapabilityValue) MeteredString(
	interpreter *Interpreter,
	seenReferences SeenReferences,
	locationRange LocationRange,
) string {
	common.UseMemory(interpreter, common.PathCapabilityValueStringMemoryUsage)

	borrowType := v.BorrowType
	if borrowType == nil {
		return fmt.Sprintf(
			"Capability(address: %s, path: %s)",
			v.address.MeteredString(interpreter, seenReferences, locationRange),
			v.Path.MeteredString(interpreter, seenReferences, locationRange),
		)
	} else {
		return fmt.Sprintf(
			"Capability<%s>(address: %s, path: %s)",
			borrowType.String(),
			v.address.MeteredString(interpreter, seenReferences, locationRange),
			v.Path.MeteredString(interpreter, seenReferences, locationRange),
		)
	}
}

func (v *PathCapabilityValue) newBorrowFunction(
	interpreter *Interpreter,
	borrowType *sema.ReferenceType,
) BoundFunctionValue {
	return NewBoundHostFunctionValue(
		interpreter,
		v,
		sema.CapabilityTypeBorrowFunctionType(borrowType),
		func(_ Invocation) Value {
			// Borrowing is never allowed
			return Nil
		},
	)
}

func (v *PathCapabilityValue) newCheckFunction(
	interpreter *Interpreter,
	borrowType *sema.ReferenceType,
) BoundFunctionValue {
	return NewBoundHostFunctionValue(
		interpreter,
		v,
		sema.CapabilityTypeCheckFunctionType(borrowType),
		func(_ Invocation) Value {
			// Borrowing is never allowed
			return FalseValue
		},
	)
}

func (v *PathCapabilityValue) GetMember(interpreter *Interpreter, _ LocationRange, name string) Value {
	switch name {
	case sema.CapabilityTypeBorrowFunctionName:
		var borrowType *sema.ReferenceType
		if v.BorrowType != nil {
			// this function will panic already if this conversion fails
			borrowType, _ = interpreter.MustConvertStaticToSemaType(v.BorrowType).(*sema.ReferenceType)
		}
		return v.newBorrowFunction(interpreter, borrowType)

	case sema.CapabilityTypeCheckFunctionName:
		var borrowType *sema.ReferenceType
		if v.BorrowType != nil {
			// this function will panic already if this conversion fails
			borrowType, _ = interpreter.MustConvertStaticToSemaType(v.BorrowType).(*sema.ReferenceType)
		}
		return v.newCheckFunction(interpreter, borrowType)

	case sema.CapabilityTypeAddressFieldName:
		return v.address

	case sema.CapabilityTypeIDFieldName:
		return InvalidCapabilityID
	}

	return nil
}

func (*PathCapabilityValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	panic(errors.NewUnreachableError())
}

func (*PathCapabilityValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	panic(errors.NewUnreachableError())
}

func (v *PathCapabilityValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v *PathCapabilityValue) Equal(interpreter *Interpreter, locationRange LocationRange, other Value) bool {
	otherCapability, ok := other.(*PathCapabilityValue)
	if !ok {
		return false
	}

	// BorrowType is optional

	if v.BorrowType == nil {
		if otherCapability.BorrowType != nil {
			return false
		}
	} else if !v.BorrowType.Equal(otherCapability.BorrowType) {
		return false
	}

	return otherCapability.address.Equal(interpreter, locationRange, v.address) &&
		otherCapability.Path.Equal(interpreter, locationRange, v.Path)
}

func (*PathCapabilityValue) IsStorable() bool {
	return true
}

func (v *PathCapabilityValue) Storable(
	storage atree.SlabStorage,
	address atree.Address,
	maxInlineSize uint64,
) (atree.Storable, error) {
	return maybeLargeImmutableStorable(
		v,
		storage,
		address,
		maxInlineSize,
	)
}

func (*PathCapabilityValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (*PathCapabilityValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v *PathCapabilityValue) Transfer(
	interpreter *Interpreter,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) Value {
	if remove {
		v.DeepRemove(interpreter, true)
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (v *PathCapabilityValue) Clone(interpreter *Interpreter) Value {
	return &PathCapabilityValue{
		BorrowType: v.BorrowType,
		Path:       v.Path.Clone(interpreter).(PathValue),
		address:    v.address.Clone(interpreter).(AddressValue),
	}
}

func (v *PathCapabilityValue) DeepRemove(interpreter *Interpreter, _ bool) {
	v.address.DeepRemove(interpreter, false)
	v.Path.DeepRemove(interpreter, false)
}

func (v *PathCapabilityValue) ByteSize() uint32 {
	return mustStorableSize(v)
}

func (v *PathCapabilityValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (v *PathCapabilityValue) ChildStorables() []atree.Storable {
	return []atree.Storable{
		v.address,
		v.Path,
	}
}

func (v *PathCapabilityValue) AddressPath() AddressPath {
	return AddressPath{
		Address: common.Address(v.address),
		Path:    v.Path,
	}
}

// NOTE: NEVER change, only add/increment; ensure uint64
const (
	// encodedPathCapabilityValueAddressFieldKey    uint64 = 0
	// encodedPathCapabilityValuePathFieldKey       uint64 = 1
	// encodedPathCapabilityValueBorrowTypeFieldKey uint64 = 2

	// !!! *WARNING* !!!
	//
	// encodedPathCapabilityValueLength MUST be updated when new element is added.
	// It is used to verify encoded capability length during decoding.
	encodedPathCapabilityValueLength = 3
)

// Encode encodes PathCapabilityValue as
//
//	cbor.Tag{
//				Number: CBORTagPathCapabilityValue,
//				Content: []any{
//						encodedPathCapabilityValueAddressFieldKey:    AddressValue(v.Address),
//						encodedPathCapabilityValuePathFieldKey:       PathValue(v.Path),
//						encodedPathCapabilityValueBorrowTypeFieldKey: StaticType(v.BorrowType),
//					},
//	}
func (v *PathCapabilityValue) Encode(e *atree.Encoder) error {
	// Encode tag number and array head
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagPathCapabilityValue,
		// array, 3 items follow
		0x83,
	})
	if err != nil {
		return err
	}

	// Encode address at array index encodedPathCapabilityValueAddressFieldKey
	err = v.address.Encode(e)
	if err != nil {
		return err
	}

	// Encode path at array index encodedPathCapabilityValuePathFieldKey
	err = v.Path.Encode(e)
	if err != nil {
		return err
	}

	// Encode borrow type at array index encodedPathCapabilityValueBorrowTypeFieldKey

	if v.BorrowType == nil {
		return e.CBOR.EncodeNil()
	} else {
		return v.BorrowType.Encode(e.CBOR)
	}
}

func (v *PathCapabilityValue) Address() AddressValue {
	return v.address
}
