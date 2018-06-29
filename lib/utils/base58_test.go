package utils

import (
	"bytes"

	"testing"
)

func getTextCases() map[string][]byte {
	cases := map[string][]byte{
		"1JRm1sbAPxJzeu3GQLobsFErnwwneRhahq": []byte{0, 191, 40, 221, 1, 240, 219, 33, 120, 128, 170, 129, 118, 165, 252, 95, 202, 215, 142, 182, 201, 79, 183, 239, 4},
		"18bQ58K4fntBRbqBaRpuc6YSjkbectzjxS": []byte{0, 83, 74, 72, 49, 109, 133, 99, 91, 117, 208, 184, 133, 5, 44, 131, 144, 183, 45, 86, 95, 220, 227, 125, 175},
	}
	return cases
}
func TestBase58Encode(t *testing.T) {
	cases := getTextCases()
	for expected, input := range cases {

		if result := Base58Encode(input); string(result) != expected {

			t.Fatalf("Got %s, expected %s.", string(result), expected)
		}
	}
}

func TestBase58Decode(t *testing.T) {
	cases := getTextCases()
	for input, expected := range cases {
		if result := Base58Decode([]byte(input)); !bytes.Equal(result, expected) {

			t.Fatal("Got ", result, "expected ", expected)
		}
	}
}
