package utils

import "testing"

func TestParseEndpoint(t *testing.T) {
	tests := []struct {
		desc          string
		endpoint      string
		expectedProto string
		expectedAddr  string
		expectedErr   bool
	}{
		{
			desc:        "invalid endpoint",
			endpoint:    "udp:///provider/azure.sock",
			expectedErr: true,
		},
		{
			desc:          "invalid unix endpoint",
			endpoint:      "unix://",
			expectedProto: "",
			expectedAddr:  "",
			expectedErr:   true,
		},
		{
			desc:          "valid unix endpoint",
			endpoint:      "unix:///provider/azure.sock",
			expectedProto: "unix",
			expectedAddr:  "/provider/azure.sock",
			expectedErr:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			proto, addr, err := ParseEndpoint(test.endpoint)
			if test.expectedErr && err == nil || !test.expectedErr && err != nil {
				t.Fatalf("expected error: %v, got error: %v", test.expectedErr, err)
			}
			if proto != test.expectedProto {
				t.Fatalf("expected proto: %v, got: %v", test.expectedProto, proto)
			}
			if addr != test.expectedAddr {
				t.Fatalf("expected addr: %v, got: %v", test.expectedAddr, addr)
			}
		})
	}
}

func TestGetGRPCMethodName(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectedOutput string
	}{
		{
			name:           "With_Correct_Method_Name",
			input:          "/v1beta1.KeyManagementService/Encrypt",
			expectedOutput: "encrypt",
		},
		{
			name:           "With_Incorrect_Method_Name",
			input:          "/Encrypt",
			expectedOutput: "unknown",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			methodName := getGRPCMethodName(testCase.input)

			if methodName != testCase.expectedOutput {
				t.Fatalf("expected output: '%s', found: '%s'", testCase.expectedOutput, methodName)
			}
		})
	}
}
