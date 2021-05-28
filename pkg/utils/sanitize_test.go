package utils

import "testing"

func TestSanitizeString(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectedOutput string
	}{
		{
			name:           "With_White_Spaces",
			input:          " hello ",
			expectedOutput: "hello",
		},
		{
			name:           "With_Double_Quotes",
			input:          "\"hello\"",
			expectedOutput: "hello",
		},
		{
			name:           "With_White_Spaces_And_Double_Quotes",
			input:          " \"hello\" ",
			expectedOutput: "hello",
		},
		{
			name:           "With_Double_Quotes_And_White_Spaces",
			input:          "\" hello \"",
			expectedOutput: "hello",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			sanitizedString := SanitizeString(testCase.input)

			if sanitizedString != testCase.expectedOutput {
				t.Fatalf("expected output: '%s', found: '%s'", testCase.expectedOutput, sanitizedString)
			}
		})
	}
}
