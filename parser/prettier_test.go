package parser

import (
	"github.com/onflow/cadence/ast"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestPrettyFunctionDeclaration(t *testing.T) {

	t.Run("with spaces, empty lines, line comments for params", func(t *testing.T) {
		testPretty(t, `
// Random comment

// Function multiply
fun multiply(

		// first param
	_ x: Int,

		// second param
	_ y: Int,
): Int {
		// multiplies the params

    return x * y // multiply
}
`,
			// TODO(output-comments): the closing args parenthesis should be in new line when there are comments or newlines between params?
			// TODO(output-comments): omit the ending spaces when args are printed in separate lines due to comments
			// TODO(output-comments): attach comments to expressions
			`
// Random comment

// Function multiply
fun multiply(
	// first param
	_ x: Int,
	// second param
	_ y: Int,
): Int {
	// multiplies the params
    return x * y // multiply
}
`)
	})

	t.Run("multi declarations, with access", func(t *testing.T) {
		testPretty(t, `
// Function hello
fun hello() {}

// Random comment


access(all)
// Function bye
fun bye() {}
`, `
// Function hello
fun hello() {}

// Random comment
access(all)
// Function bye
fun bye() {}
`)
	})
}

func TestPrettyVariableDeclaration(t *testing.T) {
	t.Run("line comment", func(t *testing.T) {
		testPretty(t, `
// foo
let foo: @AB = x
`, `
// foo
let foo: @AB = x`)
	})

	t.Run("line comment, with access", func(t *testing.T) {
		testPretty(t, `
// foo
access(all)
let foo: @AB = x
`, `
// foo
access(all)
let foo: @AB = x`)
	})
}

func TestPrettyContractDeclaration(t *testing.T) {
	t.Run("line comment", func(t *testing.T) {
		testPretty(t, `
/// FooBar contract
///
access(all)
contract FooBar : Foo, Bar {}
`, `
/// FooBar contract
///
access(all)
contract FooBar : Foo, Bar {}
`)
	})
}

func TestPrettyEventDeclaration(t *testing.T) {
	t.Run("line doc comment", func(t *testing.T) {
		testPretty(t, `
/// Hello event
access(all)
event Hello()
`, `
/// Hello event
access(all)
event Hello()
`)
	})

	t.Run("param line comments", func(t *testing.T) {
		testPretty(t, `
/// Hello event
access(all)
event Hello(
	// a
	a: String,

	// b
	b: String,
)
`, `
/// Hello event
access(all)
event Hello(
	// a
	a: String,
	// b
	b: String,
)
`)
	})

	t.Run("param line comments", func(t *testing.T) {
		testPretty(t, `
/// Hello event
access(all)
event Hello(/* before a */ a: String, /* before b */ b: String)
`, `
/// Hello event
access(all)
event Hello(/* before a */ a: String, /* before b */ b: String)
`)
	})
}

func TestPrettyFunctionInvocation(t *testing.T) {
	t.Run("line comments", func(t *testing.T) {
		testPretty(t, `
// say hello
FooBar.hello()
`, `
// say hello
FooBar.hello()
`)
	})
}

func TestPrettyVariableAssignment(t *testing.T) {
	t.Run("line comments", func(t *testing.T) {
		testPretty(t, `
// test message
message = "Hello"
`, `
// test message
message = "Hello"
`)
	})
}

func TestPrettyIfStatement(t *testing.T) {
	t.Run("line comments", func(t *testing.T) {
		testPretty(t, `
// before if
if someCondition {
	// noop
}
`, `
// before if
if someCondition {
	// noop
}
`)
	})
}

func TestPrettyForStatement(t *testing.T) {
	t.Run("line comments", func(t *testing.T) {
		testPretty(t, `
// before for
for x in y {
	// noop
}
`, `
// before for
for x in y {
	// noop
}
`)
	})
}

func TestPrettyCommentingPatterns(t *testing.T) {
	t.Run("large section markers with spacing", func(t *testing.T) {
		testPretty(t, `
/**************************
	  	Welcome
**************************/

/// Hello event
event Hello()

/**************************
	  	Goodbye
**************************/

/// Bye event
event Bye()
`, `
/**************************
	  	Welcome
**************************/

/// Hello event
event Hello()

/**************************
	  	Goodbye
**************************/

/// Bye event
event Bye()
`)
	})

	t.Run("inline section markers", func(t *testing.T) {
		testPretty(t, `
/* Welcome */
//
// Hello event
event Hello()
`, `
/* Welcome */
//
// Hello event
event Hello()
`)
	})
}

func testPretty(t *testing.T, source, expected string) {
	// trim ending and trailing spaces for easier use with template strings
	s := strings.TrimSpace(source)
	e := strings.TrimSpace(expected)

	program, err := ParseProgram(nil, []byte(s), Config{})
	require.NoError(t, err)
	require.Equal(t, e, ast.Prettier(program))
}
