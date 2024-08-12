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

package capcons

import (
	"github.com/onflow/cadence/migrations"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type CapabilityMigrationReporter interface {
	MigratedPathCapability(
		accountAddress common.Address,
		addressPath interpreter.AddressPath,
		borrowType *interpreter.ReferenceStaticType,
		capabilityID interpreter.UInt64Value,
	)
	MissingCapabilityID(
		accountAddress common.Address,
		addressPath interpreter.AddressPath,
	)
	MissingBorrowType(
		accountAddress common.Address,
		addressPath interpreter.AddressPath,
	)
}

// CapabilityValueMigration migrates all path capabilities to ID capabilities,
// using the path to ID capability controller mapping generated by LinkValueMigration.
type CapabilityValueMigration struct {
	PrivatePublicCapabilityMapping      *PathCapabilityMapping
	StorageCapabilityMapping            *PathTypeCapabilityMapping
	StorageCapabilityWithoutTypeMapping *PathCapabilityMapping
	Reporter                            CapabilityMigrationReporter
}

var _ migrations.ValueMigration = &CapabilityValueMigration{}

func (*CapabilityValueMigration) Name() string {
	return "CapabilityValueMigration"
}

func (*CapabilityValueMigration) Domains() map[string]struct{} {
	return nil
}

var fullyEntitledAccountReferenceStaticType = interpreter.ConvertSemaReferenceTypeToStaticReferenceType(
	nil,
	sema.FullyEntitledAccountReferenceType,
)

// Migrate migrates a path capability to an ID capability in the given value.
// If a value is returned, the value must be updated with the replacement in the parent.
// If nil is returned, the value was not updated and no operation has to be performed.
func (m *CapabilityValueMigration) Migrate(
	storageKey interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	value interpreter.Value,
	_ *interpreter.Interpreter,
	_ migrations.ValueMigrationPosition,
) (
	interpreter.Value,
	error,
) {

	// Migrate path capabilities to ID capabilities
	if pathCapabilityValue, ok := value.(*interpreter.PathCapabilityValue); ok { //nolint:staticcheck
		return m.migratePathCapabilityValue(pathCapabilityValue, storageKey)
	}

	return nil, nil
}

func (m *CapabilityValueMigration) migratePathCapabilityValue(
	oldCapability *interpreter.PathCapabilityValue, //nolint:staticcheck
	storageKey interpreter.StorageKey,
) (interpreter.Value, error) {

	reporter := m.Reporter

	capabilityAddressPath := oldCapability.AddressPath()

	oldBorrowType := oldCapability.BorrowType

	var capabilityID interpreter.UInt64Value
	var controllerBorrowType *interpreter.ReferenceStaticType

	targetPath := capabilityAddressPath.Path
	switch targetPath.Domain {
	case common.PathDomainPrivate, common.PathDomainPublic:
		var ok bool
		capabilityID, controllerBorrowType, ok = m.PrivatePublicCapabilityMapping.Get(capabilityAddressPath)
		if !ok {
			if reporter != nil {
				reporter.MissingCapabilityID(
					storageKey.Address,
					capabilityAddressPath,
				)
			}
			return nil, nil
		}

		// Convert untyped path capability value to typed ID capability value
		// by using capability controller's borrow type
		if oldBorrowType == nil {
			oldBorrowType = controllerBorrowType
		}

	case common.PathDomainStorage:

		// Cannot migrate storage capabilities without a borrow type yet
		if oldBorrowType != nil {
			var ok bool
			capabilityID, ok = m.StorageCapabilityMapping.Get(capabilityAddressPath, oldBorrowType.ID())
			if !ok {
				if reporter != nil {
					reporter.MissingCapabilityID(
						storageKey.Address,
						capabilityAddressPath,
					)
				}
				return nil, nil
			}
		} else {
			var ok bool
			capabilityID, oldBorrowType, ok = m.StorageCapabilityWithoutTypeMapping.Get(capabilityAddressPath)
			if !ok {
				if reporter != nil {
					reporter.MissingCapabilityID(
						storageKey.Address,
						capabilityAddressPath,
					)
				}
				return nil, nil
			}
		}

	default:
		panic(errors.NewUnexpectedError("unexpected path domain: %s", targetPath.Domain))
	}

	newBorrowType, ok := oldBorrowType.(*interpreter.ReferenceStaticType)
	if !ok {
		panic(errors.NewUnexpectedError("unexpected non-reference borrow type: %T", oldBorrowType))
	}

	newCapability := interpreter.NewUnmeteredCapabilityValue(
		capabilityID,
		oldCapability.Address,
		newBorrowType,
	)

	if reporter != nil {
		reporter.MigratedPathCapability(
			storageKey.Address,
			capabilityAddressPath,
			newBorrowType,
			capabilityID,
		)
	}

	return newCapability, nil
}

func (m *CapabilityValueMigration) CanSkip(valueType interpreter.StaticType) bool {
	return CanSkipCapabilityValueMigration(valueType)
}

func CanSkipCapabilityValueMigration(valueType interpreter.StaticType) bool {
	switch valueType := valueType.(type) {
	case *interpreter.DictionaryStaticType:
		return CanSkipCapabilityValueMigration(valueType.KeyType) &&
			CanSkipCapabilityValueMigration(valueType.ValueType)

	case interpreter.ArrayStaticType:
		return CanSkipCapabilityValueMigration(valueType.ElementType())

	case *interpreter.OptionalStaticType:
		return CanSkipCapabilityValueMigration(valueType.Type)

	case *interpreter.CapabilityStaticType:
		return false

	case interpreter.PrimitiveStaticType:

		switch valueType {
		case interpreter.PrimitiveStaticTypeCapability:
			return false

		case interpreter.PrimitiveStaticTypeBool,
			interpreter.PrimitiveStaticTypeVoid,
			interpreter.PrimitiveStaticTypeAddress,
			interpreter.PrimitiveStaticTypeMetaType,
			interpreter.PrimitiveStaticTypeBlock,
			interpreter.PrimitiveStaticTypeString,
			interpreter.PrimitiveStaticTypeCharacter:

			return true
		}

		if !valueType.IsDeprecated() { //nolint:staticcheck
			semaType := valueType.SemaType()

			if sema.IsSubType(semaType, sema.NumberType) ||
				sema.IsSubType(semaType, sema.PathType) {

				return true
			}
		}
	}

	return false
}
