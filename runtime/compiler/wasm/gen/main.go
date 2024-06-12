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

package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"
)

const fileTemplate = `// Code generated by utils/version. DO NOT EDIT.
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

{{goGenerateComment}}

package wasm

import (
	"io"
)

{{range .Instructions -}}
// Instruction{{.Identifier}} is the '{{.Name}}' instruction
//
type Instruction{{.Identifier}} struct{{if .Arguments}} {
{{- range .Arguments}}{{- if .Identifier}}
	{{.Identifier}} {{.Type.FieldType}}{{end}}{{end}}
}
{{- else}}{}{{- end}}

func (Instruction{{.Identifier}}) isInstruction() {}

func (i Instruction{{.Identifier}}) write(w *WASMWriter) error {
	err := w.writeOpcode({{.OpcodeList}})
	if err != nil {
		return err
	}
{{range .Arguments}}
	{{.Variable}} := i.{{.Identifier}}
	{{.Type.Write .Variable}}
{{end}}
	return nil
}

{{end -}}

const (
{{- range .Instructions }}
	// {{.OpcodeIdentifier}} is the opcode for the '{{.Name}}' instruction
	{{.OpcodeIdentifier}} opcode = {{.Opcode | printf "0x%x"}}
{{- end}}
)

// readInstruction reads an instruction in the WASM binary
//
func (r *WASMReader) readInstruction() (Instruction, error) {
	opcodeOffset := r.buf.offset
	b, err := r.buf.ReadByte()

	c := opcode(b)

	if err != nil {
		if err == io.EOF {
			return nil, MissingEndInstructionError{
				Offset: int(opcodeOffset),
			}
		} else {
			return nil, InvalidOpcodeError{
				Offset:    int(opcodeOffset),
				Opcode:    c,
				ReadError: err,
			}
		}
	}
{{switch .}}
}
`

const switchTemplate = `
switch c {
{{- range $key, $group := . }}
case {{ $key }}:
{{- if (eq (len $group.Instructions) 1)}}
{{- with (index $group.Instructions 0) }}
{{- range .Arguments}}
	{{.Type.Read .Variable}}
{{end}}
	return Instruction{{.Identifier}}{{if .Arguments}}{
{{- range .Arguments}}
		{{.Identifier}}: {{.Variable}},{{end}}
	}
{{- else}}{}{{- end}}, nil
{{end}}
{{- else}}
{{switch $group}}
{{- end}}{{end}}
default:
	return nil, InvalidOpcodeError{
		Offset:    int(opcodeOffset),
		Opcode:    c,
		ReadError: err,
	}
}
`

type opcodes []byte

type argumentType interface {
	isArgumentType()
	FieldType() string
	Read(variable string) string
	Write(variable string) string
}

type ArgumentTypeUint32 struct{}

func (t ArgumentTypeUint32) isArgumentType() {}

func (t ArgumentTypeUint32) FieldType() string {
	return "uint32"
}

func (t ArgumentTypeUint32) Read(variable string) string {
	return fmt.Sprintf(
		`%s, err := r.readUint32LEB128InstructionArgument()
	if err != nil {
		return nil, err
	}`,
		variable,
	)
}

func (t ArgumentTypeUint32) Write(variable string) string {
	return fmt.Sprintf(
		`err = w.buf.writeUint32LEB128(%s)
	if err != nil {
		return err
	}`,
		variable,
	)
}

type ArgumentTypeInt32 struct{}

func (t ArgumentTypeInt32) isArgumentType() {}

func (t ArgumentTypeInt32) FieldType() string {
	return "int32"
}

func (t ArgumentTypeInt32) Read(variable string) string {
	return fmt.Sprintf(
		`%s, err := r.readInt32LEB128InstructionArgument()
	if err != nil {
		return nil, err
	}`,
		variable,
	)
}

func (t ArgumentTypeInt32) Write(variable string) string {
	return fmt.Sprintf(
		`err = w.buf.writeInt32LEB128(%s)
	if err != nil {
		return err
	}`,
		variable,
	)
}

type ArgumentTypeInt64 struct{}

func (t ArgumentTypeInt64) isArgumentType() {}

func (t ArgumentTypeInt64) FieldType() string {
	return "int64"
}

func (t ArgumentTypeInt64) Read(variable string) string {
	return fmt.Sprintf(
		`%s, err := r.readInt64LEB128InstructionArgument()
	if err != nil {
		return nil, err
	}`,
		variable,
	)
}

func (t ArgumentTypeInt64) Write(variable string) string {
	return fmt.Sprintf(
		`err = w.buf.writeInt64LEB128(%s)
	if err != nil {
		return err
	}`,
		variable,
	)
}

type ArgumentTypeBlock struct {
	AllowElse bool
}

func (t ArgumentTypeBlock) isArgumentType() {}

func (t ArgumentTypeBlock) FieldType() string {
	return "Block"
}

func (t ArgumentTypeBlock) Read(variable string) string {
	return fmt.Sprintf(
		`%s, err := r.readBlockInstructionArgument(%v)
	if err != nil {
		return nil, err
	}`,
		variable,
		t.AllowElse,
	)
}

func (t ArgumentTypeBlock) Write(variable string) string {
	return fmt.Sprintf(
		`err = w.writeBlockInstructionArgument(%s, %v)
	if err != nil {
		return err
	}`,
		variable,
		t.AllowElse,
	)
}

type ArgumentTypeVector struct {
	ArgumentType argumentType
}

func (t ArgumentTypeVector) isArgumentType() {}

func (t ArgumentTypeVector) FieldType() string {
	return fmt.Sprintf("[]%s", t.ArgumentType.FieldType())
}

func (t ArgumentTypeVector) Read(variable string) string {
	// TODO: improve error

	elementVariable := variable + "Element"

	return fmt.Sprintf(
		`%[1]sCountOffset := r.buf.offset
	%[1]sCount, err := r.buf.readUint32LEB128()
	if err != nil {
		return nil, InvalidInstructionVectorArgumentCountError{
			Offset: int(%[1]sCountOffset),
			ReadError: err,
		}
	}

	%[1]s := make(%[2]s, %[1]sCount)

	for i := uint32(0); i < %[1]sCount; i++ {
		%[3]s
		%[1]s[i] = %[4]s
	}`,
		variable,
		t.FieldType(),
		t.ArgumentType.Read(elementVariable),
		elementVariable,
	)
}

func (t ArgumentTypeVector) Write(variable string) string {
	// TODO: improve error

	elementVariable := variable + "Element"

	return fmt.Sprintf(
		`%[1]sCount := len(%[1]s)
	err = w.buf.writeUint32LEB128(uint32(%[1]sCount))
	if err != nil {
		return err
	}

	for i := 0; i < %[1]sCount; i++ {
		%[3]s := %[1]s[i]
		%[2]s
	}`,
		variable,
		t.ArgumentType.Write(elementVariable),
		elementVariable,
	)
}

type argument struct {
	Type       argumentType
	Identifier string
}

func (a argument) Variable() string {
	first := strings.ToLower(string(a.Identifier[0]))
	rest := a.Identifier[1:]
	return first + rest
}

type arguments []argument

type instruction struct {
	Name      string
	Opcodes   opcodes
	Arguments arguments
}

var identifierPartRegexp = regexp.MustCompile("(^|[._])[A-Za-z0-9]")

func (ins instruction) Identifier() string {
	return string(identifierPartRegexp.ReplaceAllFunc([]byte(ins.Name), func(bytes []byte) []byte {
		return []byte(strings.ToUpper(string(bytes[len(bytes)-1])))
	}))
}
func (ins instruction) OpcodeList() string {
	var b strings.Builder

	count := len(ins.Opcodes)

	// prefix
	for i := 0; i < count-1; i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		opcode := ins.Opcodes[i]
		_, err := fmt.Fprintf(&b, "0x%x", opcode)
		if err != nil {
			panic(err)
		}
	}

	// final opcode
	if count > 1 {
		b.WriteString(", ")
	}
	_, err := b.WriteString(ins.OpcodeIdentifier())
	if err != nil {
		panic(err)
	}

	return b.String()
}

func (ins instruction) Opcode() byte {
	return ins.Opcodes[len(ins.Opcodes)-1]
}

func (ins instruction) OpcodeIdentifier() string {
	return fmt.Sprintf("opcode%s", ins.Identifier())
}

type instructionGroup struct {
	Instructions []instruction
	Depth        int
}

func (group instructionGroup) GroupByOpcode() map[string]instructionGroup {
	result := map[string]instructionGroup{}

	for _, ins := range group.Instructions {
		innerDepth := group.Depth + 1
		atEnd := len(ins.Opcodes) <= innerDepth
		opcode := ins.Opcodes[group.Depth]
		var key string
		if atEnd {
			key = ins.OpcodeIdentifier()
		} else {
			key = fmt.Sprintf("0x%x", opcode)
		}
		innerGroup := result[key]
		innerGroup.Depth = innerDepth
		innerGroup.Instructions = append(innerGroup.Instructions, ins)
		result[key] = innerGroup
	}

	return result
}

var trailingWhitespaceRegexp = regexp.MustCompile("(?m:[ \t]+$)")

const target = "instructions.go"

var indexArgumentType = ArgumentTypeUint32{}

func main() {

	f, err := os.Create(target)
	if err != nil {
		panic(fmt.Errorf("could not create %s: %w\n", target, err))
	}
	defer func() {
		_ = f.Close()
	}()

	var generateSwitch func(group instructionGroup) (string, error)

	templateFuncs := map[string]any{
		"goGenerateComment": func() string {
			// NOTE: must be templated/injected, as otherwise
			// it will be detected itself as a go generate invocation itself
			return "//go:generate go run ./gen/main.go\n//go:generate go fmt $GOFILE"
		},
		"switch": func(group instructionGroup) (string, error) {
			res, err := generateSwitch(group)
			if err != nil {
				return "", err
			}
			pad := strings.Repeat("\t", group.Depth+1)
			padded := pad + strings.ReplaceAll(res, "\n", "\n"+pad)
			trimmed := trailingWhitespaceRegexp.ReplaceAll([]byte(padded), nil)
			return string(trimmed), nil
		},
	}

	parsedSwitchTemplate := template.Must(
		template.New("switch").
			Funcs(templateFuncs).
			Parse(switchTemplate),
	)

	parsedFileTemplate := template.Must(
		template.New("instructions").
			Funcs(templateFuncs).
			Parse(fileTemplate),
	)

	generateSwitch = func(instructions instructionGroup) (string, error) {
		var b strings.Builder
		err := parsedSwitchTemplate.Execute(&b, instructions.GroupByOpcode())
		if err != nil {
			return "", err
		}
		return b.String(), nil
	}

	declare := func(instructions []instruction) {
		err = parsedFileTemplate.Execute(f,
			instructionGroup{
				Depth:        0,
				Instructions: instructions,
			},
		)
		if err != nil {
			panic(err)
		}
	}

	declare([]instruction{
		// Control Instructions
		{
			Name:      "unreachable",
			Opcodes:   opcodes{0x0},
			Arguments: arguments{},
		},
		{
			Name:      "nop",
			Opcodes:   opcodes{0x01},
			Arguments: arguments{},
		},
		{
			Name:    "block",
			Opcodes: opcodes{0x02},
			Arguments: arguments{
				{Identifier: "Block", Type: ArgumentTypeBlock{AllowElse: false}},
			},
		},
		{
			Name:    "loop",
			Opcodes: opcodes{0x03},
			Arguments: arguments{
				{Identifier: "Block", Type: ArgumentTypeBlock{AllowElse: false}},
			},
		},
		{
			Name:    "if",
			Opcodes: opcodes{0x04},
			Arguments: arguments{
				{Identifier: "Block", Type: ArgumentTypeBlock{AllowElse: true}},
			},
		},
		{
			Name:      "end",
			Opcodes:   opcodes{0x0B},
			Arguments: arguments{},
		},
		{
			Name:    "br",
			Opcodes: opcodes{0x0C},
			Arguments: arguments{
				{Identifier: "LabelIndex", Type: indexArgumentType},
			},
		},
		{
			Name:    "br_if",
			Opcodes: opcodes{0x0D},
			Arguments: arguments{
				{Identifier: "LabelIndex", Type: indexArgumentType},
			},
		},
		{
			Name:    "br_table",
			Opcodes: opcodes{0x0E},
			Arguments: arguments{
				{Identifier: "LabelIndices", Type: ArgumentTypeVector{ArgumentType: indexArgumentType}},
				{Identifier: "DefaultLabelIndex", Type: indexArgumentType},
			},
		},
		{
			Name:      "return",
			Opcodes:   opcodes{0x0F},
			Arguments: arguments{},
		},
		{
			Name:    "call",
			Opcodes: opcodes{0x10},
			Arguments: arguments{
				{Identifier: "FuncIndex", Type: indexArgumentType},
			},
		},
		{
			Name:    "call_indirect",
			Opcodes: opcodes{0x11},
			Arguments: arguments{
				{Identifier: "TypeIndex", Type: indexArgumentType},
				{Identifier: "TableIndex", Type: indexArgumentType},
			},
		},
		// Reference Instructions
		{
			Name:    "ref.null",
			Opcodes: opcodes{0xD0},
			Arguments: arguments{
				{Identifier: "TypeIndex", Type: indexArgumentType},
			},
		},
		{
			Name:      "ref.is_null",
			Opcodes:   opcodes{0xD1},
			Arguments: arguments{},
		},
		{
			Name:    "ref.func",
			Opcodes: opcodes{0xD2},
			Arguments: arguments{
				{Identifier: "FuncIndex", Type: indexArgumentType},
			},
		},
		// Parametric Instructions
		{
			Name:      "drop",
			Opcodes:   opcodes{0x1A},
			Arguments: arguments{},
		},
		{
			Name:      "select",
			Opcodes:   opcodes{0x1B},
			Arguments: arguments{},
		},
		// Variable Instructions
		{
			Name:    "local.get",
			Opcodes: opcodes{0x20},
			Arguments: arguments{
				{Identifier: "LocalIndex", Type: indexArgumentType},
			},
		},
		{
			Name:    "local.set",
			Opcodes: opcodes{0x21},
			Arguments: arguments{
				{Identifier: "LocalIndex", Type: indexArgumentType},
			},
		},
		{
			Name:    "local.tee",
			Opcodes: opcodes{0x22},
			Arguments: arguments{
				{Identifier: "LocalIndex", Type: indexArgumentType},
			},
		},
		{
			Name:    "global.get",
			Opcodes: opcodes{0x23},
			Arguments: arguments{
				{Identifier: "GlobalIndex", Type: indexArgumentType},
			},
		},
		{
			Name:    "global.set",
			Opcodes: opcodes{0x24},
			Arguments: arguments{
				{Identifier: "GlobalIndex", Type: indexArgumentType},
			},
		},
		// Numeric Instructions
		// const instructions are followed by the respective literal
		{
			Name:    "i32.const",
			Opcodes: opcodes{0x41},
			Arguments: arguments{
				// i32, "Uninterpreted integers are encoded as signed integers."
				{Identifier: "Value", Type: ArgumentTypeInt32{}},
			},
		},
		{
			Name:    "i64.const",
			Opcodes: opcodes{0x42},
			Arguments: arguments{
				// i64, "Uninterpreted integers are encoded as signed integers."
				{Identifier: "Value", Type: ArgumentTypeInt64{}},
			},
		},
		// All other numeric instructions are plain opcodes without any immediates.
		{
			Name:      "i32.eqz",
			Opcodes:   opcodes{0x45},
			Arguments: arguments{},
		},
		{
			Name:      "i32.eq",
			Opcodes:   opcodes{0x46},
			Arguments: arguments{},
		},
		{
			Name:      "i32.ne",
			Opcodes:   opcodes{0x47},
			Arguments: arguments{},
		},
		{
			Name:      "i32.lt_s",
			Opcodes:   opcodes{0x48},
			Arguments: arguments{},
		},
		{
			Name:      "i32.lt_u",
			Opcodes:   opcodes{0x49},
			Arguments: arguments{},
		},
		{
			Name:      "i32.gt_s",
			Opcodes:   opcodes{0x4a},
			Arguments: arguments{},
		},
		{
			Name:      "i32.gt_u",
			Opcodes:   opcodes{0x4b},
			Arguments: arguments{},
		},
		{
			Name:      "i32.le_s",
			Opcodes:   opcodes{0x4c},
			Arguments: arguments{},
		},
		{
			Name:      "i32.le_u",
			Opcodes:   opcodes{0x4d},
			Arguments: arguments{},
		},
		{
			Name:      "i32.ge_s",
			Opcodes:   opcodes{0x4e},
			Arguments: arguments{},
		},
		{
			Name:      "i32.ge_u",
			Opcodes:   opcodes{0x4f},
			Arguments: arguments{},
		},
		{
			Name:      "i64.eqz",
			Opcodes:   opcodes{0x50},
			Arguments: arguments{},
		},
		{
			Name:      "i64.eq",
			Opcodes:   opcodes{0x51},
			Arguments: arguments{},
		},
		{
			Name:      "i64.ne",
			Opcodes:   opcodes{0x52},
			Arguments: arguments{},
		},
		{
			Name:      "i64.lt_s",
			Opcodes:   opcodes{0x53},
			Arguments: arguments{},
		},
		{
			Name:      "i64.lt_u",
			Opcodes:   opcodes{0x54},
			Arguments: arguments{},
		},
		{
			Name:      "i64.gt_s",
			Opcodes:   opcodes{0x55},
			Arguments: arguments{},
		},
		{
			Name:      "i64.gt_u",
			Opcodes:   opcodes{0x56},
			Arguments: arguments{},
		},
		{
			Name:      "i64.le_s",
			Opcodes:   opcodes{0x57},
			Arguments: arguments{},
		},
		{
			Name:      "i64.le_u",
			Opcodes:   opcodes{0x58},
			Arguments: arguments{},
		},
		{
			Name:      "i64.ge_s",
			Opcodes:   opcodes{0x59},
			Arguments: arguments{},
		},
		{
			Name:      "i64.ge_u",
			Opcodes:   opcodes{0x5a},
			Arguments: arguments{},
		},

		{
			Name:      "i32.clz",
			Opcodes:   opcodes{0x67},
			Arguments: arguments{},
		},
		{
			Name:      "i32.ctz",
			Opcodes:   opcodes{0x68},
			Arguments: arguments{},
		},
		{
			Name:      "i32.popcnt",
			Opcodes:   opcodes{0x69},
			Arguments: arguments{},
		},
		{
			Name:      "i32.add",
			Opcodes:   opcodes{0x6a},
			Arguments: arguments{},
		},
		{
			Name:      "i32.sub",
			Opcodes:   opcodes{0x6b},
			Arguments: arguments{},
		},
		{
			Name:      "i32.mul",
			Opcodes:   opcodes{0x6c},
			Arguments: arguments{},
		},
		{
			Name:      "i32.div_s",
			Opcodes:   opcodes{0x6d},
			Arguments: arguments{},
		},
		{
			Name:      "i32.div_u",
			Opcodes:   opcodes{0x6e},
			Arguments: arguments{},
		},
		{
			Name:      "i32.rem_s",
			Opcodes:   opcodes{0x6f},
			Arguments: arguments{},
		},
		{
			Name:      "i32.rem_u",
			Opcodes:   opcodes{0x70},
			Arguments: arguments{},
		},
		{
			Name:      "i32.and",
			Opcodes:   opcodes{0x71},
			Arguments: arguments{},
		},
		{
			Name:      "i32.or",
			Opcodes:   opcodes{0x72},
			Arguments: arguments{},
		},
		{
			Name:      "i32.xor",
			Opcodes:   opcodes{0x73},
			Arguments: arguments{},
		},
		{
			Name:      "i32.shl",
			Opcodes:   opcodes{0x74},
			Arguments: arguments{},
		},
		{
			Name:      "i32.shr_s",
			Opcodes:   opcodes{0x75},
			Arguments: arguments{},
		},
		{
			Name:      "i32.shr_u",
			Opcodes:   opcodes{0x76},
			Arguments: arguments{},
		},
		{
			Name:      "i32.rotl",
			Opcodes:   opcodes{0x77},
			Arguments: arguments{},
		},
		{
			Name:      "i32.rotr",
			Opcodes:   opcodes{0x78},
			Arguments: arguments{},
		},
		{
			Name:      "i64.clz",
			Opcodes:   opcodes{0x79},
			Arguments: arguments{},
		},
		{
			Name:      "i64.ctz",
			Opcodes:   opcodes{0x7a},
			Arguments: arguments{},
		},
		{
			Name:      "i64.popcnt",
			Opcodes:   opcodes{0x7b},
			Arguments: arguments{},
		},
		{
			Name:      "i64.add",
			Opcodes:   opcodes{0x7c},
			Arguments: arguments{},
		},
		{
			Name:      "i64.sub",
			Opcodes:   opcodes{0x7d},
			Arguments: arguments{},
		},
		{
			Name:      "i64.mul",
			Opcodes:   opcodes{0x7e},
			Arguments: arguments{},
		},
		{
			Name:      "i64.div_s",
			Opcodes:   opcodes{0x7f},
			Arguments: arguments{},
		},
		{
			Name:      "i64.div_u",
			Opcodes:   opcodes{0x80},
			Arguments: arguments{},
		},
		{
			Name:      "i64.rem_s",
			Opcodes:   opcodes{0x81},
			Arguments: arguments{},
		},
		{
			Name:      "i64.rem_u",
			Opcodes:   opcodes{0x82},
			Arguments: arguments{},
		},
		{
			Name:      "i64.and",
			Opcodes:   opcodes{0x83},
			Arguments: arguments{},
		},
		{
			Name:      "i64.or",
			Opcodes:   opcodes{0x84},
			Arguments: arguments{},
		},
		{
			Name:      "i64.xor",
			Opcodes:   opcodes{0x85},
			Arguments: arguments{},
		},
		{
			Name:      "i64.shl",
			Opcodes:   opcodes{0x86},
			Arguments: arguments{},
		},
		{
			Name:      "i64.shr_s",
			Opcodes:   opcodes{0x87},
			Arguments: arguments{},
		},
		{
			Name:      "i64.shr_u",
			Opcodes:   opcodes{0x88},
			Arguments: arguments{},
		},
		{
			Name:      "i64.rotl",
			Opcodes:   opcodes{0x89},
			Arguments: arguments{},
		},
		{
			Name:      "i64.rotr",
			Opcodes:   opcodes{0x8a},
			Arguments: arguments{},
		},

		{
			Name:      "i32.wrap_i64",
			Opcodes:   opcodes{0xa7},
			Arguments: arguments{},
		},

		{
			Name:      "i64.extend_i32_s",
			Opcodes:   opcodes{0xac},
			Arguments: arguments{},
		},
		{
			Name:      "i64.extend_i32_u",
			Opcodes:   opcodes{0xad},
			Arguments: arguments{},
		},
	})
}
