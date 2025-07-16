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
			strings.TrimSpace(`
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
`))
	})

	t.Run("multi declarations", func(t *testing.T) {
		testPretty(t, `
// Function hello
fun hello() {}

// Random comment


// Function bye
fun bye() {}
`, strings.TrimSpace(`
// Function hello
fun hello() {}

// Random comment
// Function bye
fun bye() {}
`))
	})
}

func testPretty(t *testing.T, source, expected string) {
	program, err := ParseProgram(nil, []byte(source), Config{})
	require.NoError(t, err)
	require.Equal(t, expected, ast.Prettier(program))
}
