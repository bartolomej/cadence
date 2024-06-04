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

package format

import (
	"strings"
)

func PadLeft(value string, separator rune, minLength uint) string {
	length := uint(len(value))
	if length >= minLength {
		return value
	}
	n := int(minLength - length)

	var builder strings.Builder
	builder.Grow(n)
	for i := 0; i < n; i++ {
		builder.WriteRune(separator)
	}
	builder.WriteString(value)
	return builder.String()
}
