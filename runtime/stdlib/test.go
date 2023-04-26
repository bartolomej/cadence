/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

package stdlib

import (
	"fmt"
	"sync"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib/contracts"
)

// This is the Cadence standard library for writing tests.
// It provides the Cadence constructs (structs, functions, etc.) that are needed to
// write tests in Cadence.

const testContractTypeName = "Test"
const blockchainTypeName = "Blockchain"
const blockchainBackendTypeName = "BlockchainBackend"
const scriptResultTypeName = "ScriptResult"
const transactionResultTypeName = "TransactionResult"
const resultStatusTypeName = "ResultStatus"
const accountTypeName = "Account"
const errorTypeName = "Error"
const matcherTypeName = "Matcher"

const succeededCaseName = "succeeded"
const failedCaseName = "failed"

const transactionCodeFieldName = "code"
const transactionAuthorizerFieldName = "authorizers"
const transactionSignersFieldName = "signers"
const transactionArgsFieldName = "arguments"

const accountAddressFieldName = "address"

const matcherTestFunctionName = "test"

const addressesFieldName = "addresses"

const TestContractLocation = common.IdentifierLocation(testContractTypeName)

var testOnce sync.Once

// Deprecated: Use TestContractChecker instead
var testContractChecker *sema.Checker

// Deprecated: Use TestContractType instead
var testContractType *sema.CompositeType

// Deprecated: Use TestContractInitializerTypes
var testContractInitializerTypes []sema.Type

var testExpectFunction *interpreter.HostFunctionValue

var equalMatcherFunction *interpreter.HostFunctionValue

var beEmptyMatcherFunction *interpreter.HostFunctionValue

var haveElementCountMatcherFunction *interpreter.HostFunctionValue

var containMatcherFunction *interpreter.HostFunctionValue

var beGreaterThanMatcherFunction *interpreter.HostFunctionValue

var beLessThanMatcherFunction *interpreter.HostFunctionValue

var newMatcherFunction *interpreter.HostFunctionValue

var testNewEmulatorBlockchainFunctionType *sema.FunctionType

// TODO: nest in future testType
// Deprecated
var testEmulatorBackend *testEmulatorBackendType

func TestContractChecker() *sema.Checker {
	testOnce.Do(initTest)
	return testContractChecker
}

func TestContractType() *sema.CompositeType {
	testOnce.Do(initTest)
	return testContractType
}

func TestContractInitializerTypes() []sema.Type {
	testOnce.Do(initTest)
	return testContractInitializerTypes
}

func initTest() {
	program, err := parser.ParseProgram(
		nil,
		contracts.TestContract,
		parser.Config{},
	)
	if err != nil {
		panic(err)
	}

	activation := sema.NewVariableActivation(sema.BaseValueActivation)
	activation.DeclareValue(AssertFunction)
	activation.DeclareValue(PanicFunction)

	testContractChecker, err = sema.NewChecker(
		program,
		TestContractLocation,
		nil,
		&sema.Config{
			BaseValueActivation: activation,
			AccessCheckMode:     sema.AccessCheckModeStrict,
		},
	)
	if err != nil {
		panic(err)
	}

	err = testContractChecker.Check()
	if err != nil {
		panic(err)
	}

	variable, ok := testContractChecker.Elaboration.GetGlobalType(testContractTypeName)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	testContractType = variable.Type.(*sema.CompositeType)

	testContractInitializerTypes = make([]sema.Type, len(testContractType.ConstructorParameters))
	for i, parameter := range testContractType.ConstructorParameters {
		testContractInitializerTypes[i] = parameter.TypeAnnotation.Type
	}

	blockchainBackendInterfaceType := initBlockchainBackendInterfaceType()

	matcherType := initMatcherType()
	matcherTestFunctionType := compositeFunctionType(matcherType, matcherTestFunctionName)

	testEmulatorBackend = newTestEmulatorBackendType(blockchainBackendInterfaceType)

	// Test.expect()
	testExpectFunctionType := initTestExpectFunctionType(matcherType)
	initTestExpectFunction(testExpectFunctionType)
	testContractType.Members.Set(
		testExpectFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			testContractType,
			testExpectFunctionName,
			testExpectFunctionType,
			testExpectFunctionDocString,
		),
	)

	// Test.newMatcher()
	newMatcherFunctionType := initNewMatcherFunctionType(matcherType)
	initNewMatcherFunction(newMatcherFunctionType, matcherTestFunctionType)
	testContractType.Members.Set(
		newMatcherFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			testContractType,
			newMatcherFunctionName,
			newMatcherFunctionType,
			newMatcherFunctionDocString,
		),
	)

	// Test.equal()
	equalMatcherFunctionType := initEqualMatcherFunctionType(matcherType)
	initEqualMatcherFunction(equalMatcherFunctionType, matcherTestFunctionType)
	testContractType.Members.Set(
		equalMatcherFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			testContractType,
			equalMatcherFunctionName,
			equalMatcherFunctionType,
			equalMatcherFunctionDocString,
		),
	)

	// Test.beEmpty()
	beEmptyMatcherFunctionType := initBeEmptyMatcherFunctionType(matcherType)
	initBeEmptyMatcherFunction(beEmptyMatcherFunctionType, matcherTestFunctionType)
	testContractType.Members.Set(
		beEmptyMatcherFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			testContractType,
			beEmptyMatcherFunctionName,
			beEmptyMatcherFunctionType,
			beEmptyMatcherFunctionDocString,
		),
	)

	// Test.haveElementCount()
	haveElementCountMatcherFunctionType := initHaveElementCountMatcherFunctionType(matcherType)
	initHaveElementCountMatcherFunction(haveElementCountMatcherFunctionType, matcherTestFunctionType)
	testContractType.Members.Set(
		haveElementCountMatcherFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			testContractType,
			haveElementCountMatcherFunctionName,
			haveElementCountMatcherFunctionType,
			haveElementCountMatcherFunctionDocString,
		),
	)

	// Test.contain()
	containMatcherFunctionType := initContainMatcherFunctionType(matcherType)
	initContainMatcherFunction(containMatcherFunctionType, matcherTestFunctionType)
	testContractType.Members.Set(
		containMatcherFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			testContractType,
			containMatcherFunctionName,
			containMatcherFunctionType,
			containMatcherFunctionDocString,
		),
	)

	// Test.beGreaterThan()
	beGreaterThanMatcherFunctionType := initBeGreaterThanMatcherFunctionType(matcherType)
	initBeGreaterThanMatcherFunction(beGreaterThanMatcherFunctionType, matcherTestFunctionType)
	testContractType.Members.Set(
		beGreaterThanMatcherFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			testContractType,
			beGreaterThanMatcherFunctionName,
			beGreaterThanMatcherFunctionType,
			beGreaterThanMatcherFunctionDocString,
		),
	)

	// Test.beLessThan()
	beLessThanMatcherFunctionType := initBeLessThanMatcherFunctionType(matcherType)
	initBeLessThanMatcherFunction(beLessThanMatcherFunctionType, matcherTestFunctionType)
	testContractType.Members.Set(
		beLessThanMatcherFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			testContractType,
			beLessThanMatcherFunctionName,
			beLessThanMatcherFunctionType,
			beLessThanMatcherFunctionDocString,
		),
	)

	blockchainType, ok := testContractType.NestedTypes.Get(blockchainTypeName)
	if !ok {
		panic(typeNotFoundError(testContractTypeName, blockchainTypeName))
	}

	testNewEmulatorBlockchainFunctionType = &sema.FunctionType{
		ReturnTypeAnnotation: sema.NewTypeAnnotation(
			blockchainType,
		),
	}

	initTestContractTypeFunctions()

	// Enrich 'Test' contract elaboration with natively implemented composite types.
	// e.g: 'EmulatorBackend' type.
	testContractChecker.Elaboration.SetCompositeType(
		testEmulatorBackend.compositeType.ID(),
		testEmulatorBackend.compositeType,
	)
}

func initBlockchainBackendInterfaceType() *sema.InterfaceType {
	typ, ok := testContractType.NestedTypes.Get(blockchainBackendTypeName)
	if !ok {
		panic(typeNotFoundError(testContractTypeName, blockchainBackendTypeName))
	}

	blockchainBackendInterfaceType, ok := typ.(*sema.InterfaceType)
	if !ok {
		panic(errors.NewUnexpectedError(
			"invalid type for '%s'. expected interface",
			blockchainBackendTypeName,
		))
	}
	return blockchainBackendInterfaceType
}

func initMatcherType() *sema.CompositeType {
	typ, ok := testContractType.NestedTypes.Get(matcherTypeName)
	if !ok {
		panic(typeNotFoundError(testContractTypeName, matcherTypeName))
	}

	matcherType, ok := typ.(*sema.CompositeType)
	if !ok {
		panic(errors.NewUnexpectedError(
			"invalid type for '%s'. expected struct type",
			matcherTypeName,
		))
	}
	return matcherType
}

func initTestContractTypeFunctions() {
	// Enrich 'Test' contract with natively implemented functions

	// Test.assert()
	testContractType.Members.Set(
		testAssertFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			testContractType,
			testAssertFunctionName,
			testAssertFunctionType,
			testAssertFunctionDocString,
		),
	)

	// Test.fail()
	testContractType.Members.Set(
		testFailFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			testContractType,
			testFailFunctionName,
			testFailFunctionType,
			testFailFunctionDocString,
		),
	)

	// Test.newEmulatorBlockchain()
	testContractType.Members.Set(
		testNewEmulatorBlockchainFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			testContractType,
			testNewEmulatorBlockchainFunctionName,
			testNewEmulatorBlockchainFunctionType,
			testNewEmulatorBlockchainFunctionDocString,
		),
	)

	// Test.readFile()
	testContractType.Members.Set(
		testReadFileFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			testContractType,
			testReadFileFunctionName,
			testReadFileFunctionType,
			testReadFileFunctionDocString,
		),
	)
}

func NewTestContract(
	inter *interpreter.Interpreter,
	testFramework TestFramework,
	constructor interpreter.FunctionValue,
	invocationRange ast.Range,
) (
	*interpreter.CompositeValue,
	error,
) {
	initializerTypes := TestContractInitializerTypes()
	value, err := inter.InvokeFunctionValue(
		constructor,
		nil,
		initializerTypes,
		initializerTypes,
		invocationRange,
	)
	if err != nil {
		return nil, err
	}

	compositeValue := value.(*interpreter.CompositeValue)

	// Inject natively implemented function values
	compositeValue.Functions[testAssertFunctionName] = testAssertFunction
	compositeValue.Functions[testFailFunctionName] = testFailFunction
	compositeValue.Functions[testExpectFunctionName] = testExpectFunction
	compositeValue.Functions[testNewEmulatorBlockchainFunctionName] = testNewEmulatorBlockchainFunction(testFramework)
	compositeValue.Functions[testReadFileFunctionName] = testReadFileFunction(testFramework)

	// Inject natively implemented matchers
	compositeValue.Functions[newMatcherFunctionName] = newMatcherFunction
	compositeValue.Functions[equalMatcherFunctionName] = equalMatcherFunction
	compositeValue.Functions[beEmptyMatcherFunctionName] = beEmptyMatcherFunction
	compositeValue.Functions[haveElementCountMatcherFunctionName] = haveElementCountMatcherFunction
	compositeValue.Functions[containMatcherFunctionName] = containMatcherFunction
	compositeValue.Functions[beGreaterThanMatcherFunctionName] = beGreaterThanMatcherFunction
	compositeValue.Functions[beLessThanMatcherFunctionName] = beLessThanMatcherFunction

	return compositeValue, nil
}

func typeNotFoundError(parentType, nestedType string) error {
	return errors.NewUnexpectedError("cannot find type '%s.%s'", parentType, nestedType)
}

func memberNotFoundError(parentType, member string) error {
	return errors.NewUnexpectedError("cannot find member '%s.%s'", parentType, member)
}

func compositeFunctionType(parent *sema.CompositeType, funcName string) *sema.FunctionType {
	testFunc, ok := parent.Members.Get(funcName)
	if !ok {
		panic(memberNotFoundError(parent.Identifier, funcName))
	}

	return getFunctionTypeFromMember(testFunc, funcName)
}

func interfaceFunctionType(parent *sema.InterfaceType, funcName string) *sema.FunctionType {
	testFunc, ok := parent.Members.Get(funcName)
	if !ok {
		panic(memberNotFoundError(parent.Identifier, funcName))
	}

	return getFunctionTypeFromMember(testFunc, funcName)
}

func getFunctionTypeFromMember(funcMember *sema.Member, funcName string) *sema.FunctionType {
	functionType, ok := funcMember.TypeAnnotation.Type.(*sema.FunctionType)
	if !ok {
		panic(errors.NewUnexpectedError(
			"invalid type for '%s'. expected function type",
			funcName,
		))
	}

	return functionType
}

// Functions belonging to the 'Test' contract

// 'Test.assert' function

const testAssertFunctionDocString = `
Fails the test-case if the given condition is false, and reports a message which explains how the condition is false.
`

const testAssertFunctionName = "assert"

var testAssertFunctionType = &sema.FunctionType{
	Parameters: []sema.Parameter{
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "condition",
			TypeAnnotation: sema.NewTypeAnnotation(
				sema.BoolType,
			),
		},
		{
			Identifier: "message",
			TypeAnnotation: sema.NewTypeAnnotation(
				sema.StringType,
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.VoidType,
	),
	RequiredArgumentCount: sema.RequiredArgumentCount(1),
}

var testAssertFunction = interpreter.NewUnmeteredHostFunctionValue(
	testAssertFunctionType,
	func(invocation interpreter.Invocation) interpreter.Value {
		condition, ok := invocation.Arguments[0].(interpreter.BoolValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		var message string
		if len(invocation.Arguments) > 1 {
			messageValue, ok := invocation.Arguments[1].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}
			message = messageValue.Str
		}

		if !condition {
			panic(AssertionError{
				Message:       message,
				LocationRange: invocation.LocationRange,
			})
		}

		return interpreter.Void
	},
)

// 'Test.fail' function

const testFailFunctionDocString = `
Fails the test-case with a message.
`

const testFailFunctionName = "fail"

var testFailFunctionType = &sema.FunctionType{
	Parameters: []sema.Parameter{
		{
			Identifier: "message",
			TypeAnnotation: sema.NewTypeAnnotation(
				sema.StringType,
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.VoidType,
	),
	RequiredArgumentCount: sema.RequiredArgumentCount(0),
}

var testFailFunction = interpreter.NewUnmeteredHostFunctionValue(
	testFailFunctionType,
	func(invocation interpreter.Invocation) interpreter.Value {
		var message string
		if len(invocation.Arguments) > 0 {
			messageValue, ok := invocation.Arguments[0].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}
			message = messageValue.Str
		}

		panic(AssertionError{
			Message:       message,
			LocationRange: invocation.LocationRange,
		})
	},
)

// 'Test.expect' function

const testExpectFunctionDocString = `
Expect function tests a value against a matcher, and fails the test if it's not a match.
`

const testExpectFunctionName = "expect"

func initTestExpectFunctionType(matcherType *sema.CompositeType) *sema.FunctionType {
	typeParameter := &sema.TypeParameter{
		TypeBound: sema.AnyStructType,
		Name:      "T",
		Optional:  true,
	}

	return &sema.FunctionType{
		Parameters: []sema.Parameter{
			{
				Label:      sema.ArgumentLabelNotRequired,
				Identifier: "value",
				TypeAnnotation: sema.NewTypeAnnotation(
					&sema.GenericType{
						TypeParameter: typeParameter,
					},
				),
			},
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "matcher",
				TypeAnnotation: sema.NewTypeAnnotation(matcherType),
			},
		},
		TypeParameters: []*sema.TypeParameter{
			typeParameter,
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(
			sema.VoidType,
		),
	}
}

func initTestExpectFunction(testExpectFunctionType *sema.FunctionType) {
	testExpectFunction = interpreter.NewUnmeteredHostFunctionValue(
		testExpectFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			value := invocation.Arguments[0]

			matcher, ok := invocation.Arguments[1].(*interpreter.CompositeValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			result := invokeMatcherTest(
				inter,
				matcher,
				value,
				locationRange,
			)

			if !result {
				panic(AssertionError{})
			}

			return interpreter.Void
		},
	)
}

func invokeMatcherTest(
	inter *interpreter.Interpreter,
	matcher interpreter.MemberAccessibleValue,
	value interpreter.Value,
	locationRange interpreter.LocationRange,
) bool {
	testFunc := matcher.GetMember(
		inter,
		locationRange,
		matcherTestFunctionName,
	)

	funcValue, ok := testFunc.(interpreter.FunctionValue)
	if !ok {
		panic(errors.NewUnexpectedError(
			"invalid type for '%s'. expected function",
			matcherTestFunctionName,
		))
	}

	functionType := funcValue.FunctionType()

	testResult, err := inter.InvokeExternally(
		funcValue,
		functionType,
		[]interpreter.Value{
			value,
		},
	)

	if err != nil {
		panic(err)
	}

	result, ok := testResult.(interpreter.BoolValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return bool(result)
}

// 'Test.readFile' function

const testReadFileFunctionDocString = `
Read a local file, and return the content as a string.
`

const testReadFileFunctionName = "readFile"

var testReadFileFunctionType = &sema.FunctionType{
	Parameters: []sema.Parameter{
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "path",
			TypeAnnotation: sema.NewTypeAnnotation(
				sema.StringType,
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.StringType,
	),
}

func testReadFileFunction(testFramework TestFramework) *interpreter.HostFunctionValue {
	return interpreter.NewUnmeteredHostFunctionValue(
		testReadFileFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			pathString, ok := invocation.Arguments[0].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			content, err := testFramework.ReadFile(pathString.Str)
			if err != nil {
				panic(err)
			}

			return interpreter.NewUnmeteredStringValue(content)
		},
	)
}

// 'Test.newEmulatorBlockchain' function

const testNewEmulatorBlockchainFunctionDocString = `
Creates a blockchain which is backed by a new emulator instance.
`

const testNewEmulatorBlockchainFunctionName = "newEmulatorBlockchain"

func testNewEmulatorBlockchainFunction(testFramework TestFramework) *interpreter.HostFunctionValue {
	return interpreter.NewUnmeteredHostFunctionValue(
		testNewEmulatorBlockchainFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			// Create an `EmulatorBackend`
			emulatorBackend := testEmulatorBackend.new(
				inter,
				testFramework,
				locationRange,
			)

			// Create a 'Blockchain' struct value, that wraps the emulator backend,
			// by calling the constructor of 'Blockchain'.

			blockchainConstructor := getNestedTypeConstructorValue(
				*invocation.Self,
				blockchainTypeName,
			)

			blockchain, err := inter.InvokeExternally(
				blockchainConstructor,
				blockchainConstructor.Type,
				[]interpreter.Value{
					emulatorBackend,
				},
			)

			if err != nil {
				panic(err)
			}

			return blockchain
		},
	)
}

func getNestedTypeConstructorValue(parent interpreter.Value, typeName string) *interpreter.HostFunctionValue {
	compositeValue, ok := parent.(*interpreter.CompositeValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	constructorVar := compositeValue.NestedVariables[typeName]
	constructor, ok := constructorVar.GetValue().(*interpreter.HostFunctionValue)
	if !ok {
		panic(errors.NewUnexpectedError("invalid type for constructor"))
	}
	return constructor
}

// 'Test.NewMatcher' function.
// Constructs a matcher that test only 'AnyStruct'.
// Accepts test function that accepts subtype of 'AnyStruct'.
//
// Signature:
//    fun newMatcher<T: AnyStruct>(test: ((T): Bool)): Test.Matcher
//
// where `T` is optional, and bound to `AnyStruct`.
//
// Sample usage: `Test.newMatcher(fun (_ value: Int: Bool) { return true })`

const newMatcherFunctionDocString = `
Creates a matcher with a test function.
The test function is of type '((T): Bool)', where 'T' is bound to 'AnyStruct'.
`

const newMatcherFunctionName = "newMatcher"

func initNewMatcherFunctionType(matcherType *sema.CompositeType) *sema.FunctionType {
	typeParameter := &sema.TypeParameter{
		TypeBound: sema.AnyStructType,
		Name:      "T",
		Optional:  true,
	}

	return &sema.FunctionType{
		IsConstructor: true,
		Parameters: []sema.Parameter{
			{
				Label:      sema.ArgumentLabelNotRequired,
				Identifier: "test",
				TypeAnnotation: sema.NewTypeAnnotation(
					// Type of the 'test' function: ((T): Bool)
					&sema.FunctionType{
						Parameters: []sema.Parameter{
							{
								Label:      sema.ArgumentLabelNotRequired,
								Identifier: "value",
								TypeAnnotation: sema.NewTypeAnnotation(
									&sema.GenericType{
										TypeParameter: typeParameter,
									},
								),
							},
						},
						ReturnTypeAnnotation: sema.NewTypeAnnotation(
							sema.BoolType,
						),
					},
				),
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(matcherType),
		TypeParameters: []*sema.TypeParameter{
			typeParameter,
		},
	}
}

func initNewMatcherFunction(
	newMatcherFunctionType *sema.FunctionType,
	matcherTestFunctionType *sema.FunctionType,
) {
	newMatcherFunction = interpreter.NewUnmeteredHostFunctionValue(
		newMatcherFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			test, ok := invocation.Arguments[0].(interpreter.FunctionValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			return newMatcherWithGenericTestFunction(
				invocation,
				test,
				matcherTestFunctionType,
			)
		},
	)
}

func arrayValueToSlice(value interpreter.Value) ([]interpreter.Value, error) {
	array, ok := value.(*interpreter.ArrayValue)
	if !ok {
		return nil, errors.NewDefaultUserError("value is not an array")
	}

	result := make([]interpreter.Value, 0, array.Count())

	array.Iterate(nil, func(element interpreter.Value) (resume bool) {
		result = append(result, element)
		return true
	})

	return result, nil
}

// newScriptResult Creates a "ScriptResult" using the return value of the executed script.
func newScriptResult(
	inter *interpreter.Interpreter,
	returnValue interpreter.Value,
	result *ScriptResult,
) interpreter.Value {

	if returnValue == nil {
		returnValue = interpreter.Nil
	}

	// Lookup and get 'ResultStatus' enum value.
	resultStatusConstructor := getConstructor(inter, resultStatusTypeName)
	var status interpreter.Value
	if result.Error == nil {
		succeededVar := resultStatusConstructor.NestedVariables[succeededCaseName]
		status = succeededVar.GetValue()
	} else {
		failedVar := resultStatusConstructor.NestedVariables[failedCaseName]
		status = failedVar.GetValue()
	}

	errValue := newErrorValue(inter, result.Error)

	// Create a 'ScriptResult' by calling its constructor.
	scriptResultConstructor := getConstructor(inter, scriptResultTypeName)
	scriptResult, err := inter.InvokeExternally(
		scriptResultConstructor,
		scriptResultConstructor.Type,
		[]interpreter.Value{
			status,
			returnValue,
			errValue,
		},
	)

	if err != nil {
		panic(err)
	}

	return scriptResult
}

func getConstructor(inter *interpreter.Interpreter, typeName string) *interpreter.HostFunctionValue {
	resultStatusConstructorVar := inter.FindVariable(typeName)
	resultStatusConstructor, ok := resultStatusConstructorVar.GetValue().(*interpreter.HostFunctionValue)
	if !ok {
		panic(errors.NewUnexpectedError("invalid type for constructor of '%s'", typeName))
	}

	return resultStatusConstructor
}

func addressesFromValue(accountsValue interpreter.Value) []common.Address {
	accountsArray, ok := accountsValue.(*interpreter.ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	addresses := make([]common.Address, 0)

	accountsArray.Iterate(nil, func(element interpreter.Value) (resume bool) {
		address, ok := element.(interpreter.AddressValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		addresses = append(addresses, common.Address(address))

		return true
	})

	return addresses
}

func accountsFromValue(
	inter *interpreter.Interpreter,
	accountsValue interpreter.Value,
	locationRange interpreter.LocationRange,
) []*Account {

	accountsArray, ok := accountsValue.(*interpreter.ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	accounts := make([]*Account, 0)

	accountsArray.Iterate(nil, func(element interpreter.Value) (resume bool) {
		accountValue, ok := element.(interpreter.MemberAccessibleValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		account := accountFromValue(inter, accountValue, locationRange)

		accounts = append(accounts, account)

		return true
	})

	return accounts
}

func accountFromValue(
	inter *interpreter.Interpreter,
	accountValue interpreter.MemberAccessibleValue,
	locationRange interpreter.LocationRange,
) *Account {

	// Get address
	addressValue := accountValue.GetMember(
		inter,
		locationRange,
		accountAddressFieldName,
	)
	address, ok := addressValue.(interpreter.AddressValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// Get public key
	publicKeyVal, ok := accountValue.GetMember(
		inter,
		locationRange,
		sema.AccountKeyPublicKeyFieldName,
	).(interpreter.MemberAccessibleValue)

	if !ok {
		panic(errors.NewUnreachableError())
	}

	publicKey, err := NewPublicKeyFromValue(inter, locationRange, publicKeyVal)
	if err != nil {
		panic(err)
	}

	return &Account{
		Address:   common.Address(address),
		PublicKey: publicKey,
	}
}

// newTransactionResult Creates a "TransactionResult" indicating the status of the transaction execution.
func newTransactionResult(inter *interpreter.Interpreter, result *TransactionResult) interpreter.Value {
	// Lookup and get 'ResultStatus' enum value.
	resultStatusConstructor := getConstructor(inter, resultStatusTypeName)
	var status interpreter.Value
	if result.Error == nil {
		succeededVar := resultStatusConstructor.NestedVariables[succeededCaseName]
		status = succeededVar.GetValue()
	} else {
		failedVar := resultStatusConstructor.NestedVariables[failedCaseName]
		status = failedVar.GetValue()
	}

	// Create a 'TransactionResult' by calling its constructor.
	transactionResultConstructor := getConstructor(inter, transactionResultTypeName)

	errValue := newErrorValue(inter, result.Error)

	transactionResult, err := inter.InvokeExternally(
		transactionResultConstructor,
		transactionResultConstructor.Type,
		[]interpreter.Value{
			status,
			errValue,
		},
	)

	if err != nil {
		panic(err)
	}

	return transactionResult
}

func newErrorValue(inter *interpreter.Interpreter, err error) interpreter.Value {
	if err == nil {
		return interpreter.Nil
	}

	// Create a 'Error' by calling its constructor.
	errorConstructor := getConstructor(inter, errorTypeName)

	errorValue, invocationErr := inter.InvokeExternally(
		errorConstructor,
		errorConstructor.Type,
		[]interpreter.Value{
			interpreter.NewUnmeteredStringValue(err.Error()),
		},
	)

	if invocationErr != nil {
		panic(invocationErr)
	}

	return errorValue
}

// Built-in matchers

const equalMatcherFunctionName = "equal"

const equalMatcherFunctionDocString = `
Returns a matcher that succeeds if the tested value is equal to the given value.
`

func initEqualMatcherFunctionType(matcherType *sema.CompositeType) *sema.FunctionType {
	typeParameter := &sema.TypeParameter{
		TypeBound: sema.AnyStructType,
		Name:      "T",
		Optional:  true,
	}

	return &sema.FunctionType{
		IsConstructor: false,
		TypeParameters: []*sema.TypeParameter{
			typeParameter,
		},
		Parameters: []sema.Parameter{
			{
				Label:      sema.ArgumentLabelNotRequired,
				Identifier: "value",
				TypeAnnotation: sema.NewTypeAnnotation(
					&sema.GenericType{
						TypeParameter: typeParameter,
					},
				),
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(matcherType),
	}
}

func initEqualMatcherFunction(
	equalMatcherFunctionType *sema.FunctionType,
	matcherTestFunctionType *sema.FunctionType,
) {
	equalMatcherFunction = interpreter.NewUnmeteredHostFunctionValue(
		equalMatcherFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			otherValue, ok := invocation.Arguments[0].(interpreter.EquatableValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.Interpreter

			equalTestFunc := interpreter.NewHostFunctionValue(
				nil,
				matcherTestFunctionType,
				func(invocation interpreter.Invocation) interpreter.Value {

					thisValue, ok := invocation.Arguments[0].(interpreter.EquatableValue)
					if !ok {
						panic(errors.NewUnreachableError())
					}

					equal := thisValue.Equal(
						inter,
						invocation.LocationRange,
						otherValue,
					)

					return interpreter.AsBoolValue(equal)
				},
			)

			return newMatcherWithGenericTestFunction(
				invocation,
				equalTestFunc,
				matcherTestFunctionType,
			)
		},
	)
}

const beEmptyMatcherFunctionName = "beEmpty"

const beEmptyMatcherFunctionDocString = `
Returns a matcher that succeeds if the tested value is an array or dictionary,
and the tested value contains no elements.
`

func initBeEmptyMatcherFunctionType(matcherType *sema.CompositeType) *sema.FunctionType {
	return &sema.FunctionType{
		IsConstructor:        false,
		TypeParameters:       []*sema.TypeParameter{},
		Parameters:           []sema.Parameter{},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(matcherType),
	}
}

func initBeEmptyMatcherFunction(
	beEmptyMatcherFunctionType *sema.FunctionType,
	matcherTestFunctionType *sema.FunctionType,
) {
	beEmptyMatcherFunction = interpreter.NewUnmeteredHostFunctionValue(
		beEmptyMatcherFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			beEmptyTestFunc := interpreter.NewHostFunctionValue(
				nil,
				matcherTestFunctionType,
				func(invocation interpreter.Invocation) interpreter.Value {
					var isEmpty bool
					switch value := invocation.Arguments[0].(type) {
					case *interpreter.ArrayValue:
						isEmpty = value.Count() == 0
					case *interpreter.DictionaryValue:
						isEmpty = value.Count() == 0
					default:
						panic(errors.NewDefaultUserError("expected Array or Dictionary argument"))
					}

					return interpreter.AsBoolValue(isEmpty)
				},
			)

			return newMatcherWithGenericTestFunction(
				invocation,
				beEmptyTestFunc,
				matcherTestFunctionType,
			)
		},
	)
}

const haveElementCountMatcherFunctionName = "haveElementCount"

const haveElementCountMatcherFunctionDocString = `
Returns a matcher that succeeds if the tested value is an array or dictionary,
and has the given number of elements.
`

func initHaveElementCountMatcherFunctionType(matcherType *sema.CompositeType) *sema.FunctionType {
	return &sema.FunctionType{
		IsConstructor:  false,
		TypeParameters: []*sema.TypeParameter{},
		Parameters: []sema.Parameter{
			{
				Label:      sema.ArgumentLabelNotRequired,
				Identifier: "count",
				TypeAnnotation: sema.NewTypeAnnotation(
					sema.IntType,
				),
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(matcherType),
	}
}

func initHaveElementCountMatcherFunction(
	haveElementCountMatcherFunctionType *sema.FunctionType,
	matcherTestFunctionType *sema.FunctionType,
) {
	haveElementCountMatcherFunction = interpreter.NewUnmeteredHostFunctionValue(
		haveElementCountMatcherFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			count, ok := invocation.Arguments[0].(interpreter.IntValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			haveElementCountTestFunc := interpreter.NewHostFunctionValue(
				nil,
				matcherTestFunctionType,
				func(invocation interpreter.Invocation) interpreter.Value {
					var matchingCount bool
					switch value := invocation.Arguments[0].(type) {
					case *interpreter.ArrayValue:
						matchingCount = value.Count() == count.ToInt(invocation.LocationRange)
					case *interpreter.DictionaryValue:
						matchingCount = value.Count() == count.ToInt(invocation.LocationRange)
					default:
						panic(errors.NewDefaultUserError("expected Array or Dictionary argument"))
					}

					return interpreter.AsBoolValue(matchingCount)
				},
			)

			return newMatcherWithGenericTestFunction(
				invocation,
				haveElementCountTestFunc,
				matcherTestFunctionType,
			)
		},
	)
}

const containMatcherFunctionName = "contain"

const containMatcherFunctionDocString = `
Returns a matcher that succeeds if the tested value is an array that contains
a value that is equal to the given value, or the tested value is a dictionary
that contains an entry where the key is equal to the given value.
`

func initContainMatcherFunctionType(matcherType *sema.CompositeType) *sema.FunctionType {
	return &sema.FunctionType{
		IsConstructor:  false,
		TypeParameters: []*sema.TypeParameter{},
		Parameters: []sema.Parameter{
			{
				Label:      sema.ArgumentLabelNotRequired,
				Identifier: "element",
				TypeAnnotation: sema.NewTypeAnnotation(
					sema.AnyStructType,
				),
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(matcherType),
	}
}

func initContainMatcherFunction(
	containMatcherFunctionType *sema.FunctionType,
	matcherTestFunctionType *sema.FunctionType,
) {
	containMatcherFunction = interpreter.NewUnmeteredHostFunctionValue(
		containMatcherFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			element, ok := invocation.Arguments[0].(interpreter.EquatableValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.Interpreter

			containTestFunc := interpreter.NewHostFunctionValue(
				nil,
				matcherTestFunctionType,
				func(invocation interpreter.Invocation) interpreter.Value {
					var elementFound interpreter.BoolValue
					switch value := invocation.Arguments[0].(type) {
					case *interpreter.ArrayValue:
						elementFound = value.Contains(
							inter,
							invocation.LocationRange,
							element,
						)
					case *interpreter.DictionaryValue:
						elementFound = value.ContainsKey(
							inter,
							invocation.LocationRange,
							element,
						)
					default:
						panic(errors.NewDefaultUserError("expected Array or Dictionary argument"))
					}

					return elementFound
				},
			)

			return newMatcherWithGenericTestFunction(
				invocation,
				containTestFunc,
				matcherTestFunctionType,
			)
		},
	)
}

const beGreaterThanMatcherFunctionName = "beGreaterThan"

const beGreaterThanMatcherFunctionDocString = `
Returns a matcher that succeeds if the tested value is a number and
greater than the given number.
`

func initBeGreaterThanMatcherFunctionType(matcherType *sema.CompositeType) *sema.FunctionType {
	return &sema.FunctionType{
		IsConstructor:  false,
		TypeParameters: []*sema.TypeParameter{},
		Parameters: []sema.Parameter{
			{
				Label:      sema.ArgumentLabelNotRequired,
				Identifier: "value",
				TypeAnnotation: sema.NewTypeAnnotation(
					sema.NumberType,
				),
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(matcherType),
	}
}

func initBeGreaterThanMatcherFunction(
	beGreaterThanMatcherFunctionType *sema.FunctionType,
	matcherTestFunctionType *sema.FunctionType,
) {
	beGreaterThanMatcherFunction = interpreter.NewUnmeteredHostFunctionValue(
		beGreaterThanMatcherFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			otherValue, ok := invocation.Arguments[0].(interpreter.NumberValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.Interpreter

			beGreaterThanTestFunc := interpreter.NewHostFunctionValue(
				nil,
				matcherTestFunctionType,
				func(invocation interpreter.Invocation) interpreter.Value {
					thisValue, ok := invocation.Arguments[0].(interpreter.NumberValue)
					if !ok {
						panic(errors.NewUnreachableError())
					}

					isGreaterThan := thisValue.Greater(
						inter,
						otherValue,
						invocation.LocationRange,
					)

					return isGreaterThan
				},
			)

			return newMatcherWithGenericTestFunction(
				invocation,
				beGreaterThanTestFunc,
				matcherTestFunctionType,
			)
		},
	)
}

const beLessThanMatcherFunctionName = "beLessThan"

const beLessThanMatcherFunctionDocString = `
Returns a matcher that succeeds if the tested value is a number and
less than the given number.
`

func initBeLessThanMatcherFunctionType(matcherType *sema.CompositeType) *sema.FunctionType {
	return &sema.FunctionType{
		IsConstructor:  false,
		TypeParameters: []*sema.TypeParameter{},
		Parameters: []sema.Parameter{
			{
				Label:      sema.ArgumentLabelNotRequired,
				Identifier: "value",
				TypeAnnotation: sema.NewTypeAnnotation(
					sema.NumberType,
				),
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(matcherType),
	}
}

func initBeLessThanMatcherFunction(
	beLessThanMatcherFunctionType *sema.FunctionType,
	matcherTestFunctionType *sema.FunctionType,
) {
	beLessThanMatcherFunction = interpreter.NewUnmeteredHostFunctionValue(
		beLessThanMatcherFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			otherValue, ok := invocation.Arguments[0].(interpreter.NumberValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.Interpreter

			beLessThanTestFunc := interpreter.NewHostFunctionValue(
				nil,
				matcherTestFunctionType,
				func(invocation interpreter.Invocation) interpreter.Value {
					thisValue, ok := invocation.Arguments[0].(interpreter.NumberValue)
					if !ok {
						panic(errors.NewUnreachableError())
					}

					isLessThan := thisValue.Less(
						inter,
						otherValue,
						invocation.LocationRange,
					)

					return isLessThan
				},
			)

			return newMatcherWithGenericTestFunction(
				invocation,
				beLessThanTestFunc,
				matcherTestFunctionType,
			)
		},
	)
}

// TestFailedError

type TestFailedError struct {
	Err error
}

var _ errors.UserError = TestFailedError{}

func (TestFailedError) IsUserError() {}

func (e TestFailedError) Unwrap() error {
	return e.Err
}

func (e TestFailedError) Error() string {
	return fmt.Sprintf("test failed: %s", e.Err.Error())
}

func newMatcherWithGenericTestFunction(
	invocation interpreter.Invocation,
	testFunc interpreter.FunctionValue,
	matcherTestFunctionType *sema.FunctionType,
) interpreter.Value {

	inter := invocation.Interpreter

	staticType, ok := testFunc.StaticType(inter).(interpreter.FunctionStaticType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	parameters := staticType.Type.Parameters

	// Wrap the user provided test function with a function that validates the argument types.
	// i.e: create a closure that cast the arguments.
	//
	// e.g: convert `newMatcher(test: ((Int): Bool))` to:
	//
	//  newMatcher(fun (b: AnyStruct): Bool {
	//      return test(b as! Int)
	//  })
	//
	// Note: This argument validation is only needed if the matcher was created with a user-provided function.
	// No need to validate if the matcher is created as a matcher combinator.
	//
	matcherTestFunction := interpreter.NewUnmeteredHostFunctionValue(
		matcherTestFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			inter := invocation.Interpreter

			for i, argument := range invocation.Arguments {
				paramType := parameters[i].TypeAnnotation.Type
				argumentStaticType := argument.StaticType(inter)

				if !inter.IsSubTypeOfSemaType(argumentStaticType, paramType) {
					argumentSemaType := inter.MustConvertStaticToSemaType(argumentStaticType)

					panic(interpreter.TypeMismatchError{
						ExpectedType:  paramType,
						ActualType:    argumentSemaType,
						LocationRange: invocation.LocationRange,
					})
				}
			}

			value, err := inter.InvokeFunction(testFunc, invocation)
			if err != nil {
				panic(err)
			}

			return value
		},
	)

	matcherConstructor := getNestedTypeConstructorValue(
		*invocation.Self,
		matcherTypeName,
	)
	matcher, err := inter.InvokeExternally(
		matcherConstructor,
		matcherConstructor.Type,
		[]interpreter.Value{
			matcherTestFunction,
		},
	)

	if err != nil {
		panic(err)
	}

	return matcher
}

func TestCheckerContractValueHandler(
	checker *sema.Checker,
	declaration *ast.CompositeDeclaration,
	compositeType *sema.CompositeType,
) sema.ValueDeclaration {
	constructorType, constructorArgumentLabels := sema.CompositeLikeConstructorType(
		checker.Elaboration,
		declaration,
		compositeType,
	)

	return StandardLibraryValue{
		Name:           declaration.Identifier.Identifier,
		Type:           constructorType,
		DocString:      declaration.DocString,
		Kind:           declaration.DeclarationKind(),
		Position:       &declaration.Identifier.Pos,
		ArgumentLabels: constructorArgumentLabels,
	}
}

func NewTestInterpreterContractValueHandler(
	testFramework TestFramework,
) interpreter.ContractValueHandlerFunc {
	return func(
		inter *interpreter.Interpreter,
		compositeType *sema.CompositeType,
		constructorGenerator func(common.Address) *interpreter.HostFunctionValue,
		invocationRange ast.Range,
	) interpreter.ContractValue {

		switch compositeType.Location {
		case CryptoCheckerLocation:
			contract, err := NewCryptoContract(
				inter,
				constructorGenerator(common.ZeroAddress),
				invocationRange,
			)
			if err != nil {
				panic(err)
			}
			return contract

		case TestContractLocation:
			contract, err := NewTestContract(
				inter,
				testFramework,
				constructorGenerator(common.ZeroAddress),
				invocationRange,
			)
			if err != nil {
				panic(err)
			}
			return contract

		default:
			// During tests, imported contracts can be constructed using the constructor,
			// similar to structs. Therefore, generate a constructor function.
			return constructorGenerator(common.ZeroAddress)
		}
	}
}
