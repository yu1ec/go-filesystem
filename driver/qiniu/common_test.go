package qiniu

import "testing"

func TestSaveAsToString(t *testing.T) {
	expiredDays := 1
	testCases := []struct {
		Name     string
		Expected string
		SaveAs   SaveAs
	}{
		{
			Name:     "test_saveas_withkey",
			Expected: "saveas/dGVzdDp0ZXN0",
			SaveAs:   SaveAs{SaveBucket: "test", SaveKey: "test"},
		},
		{
			Name:     "test_saveas_withoutkey",
			Expected: "saveas/dGVzdA==",
			SaveAs:   SaveAs{SaveBucket: "test"},
		},
		{
			Name:     "test_saveas_with_deleteafterdays",
			Expected: "saveas/dGVzdDp0ZXN0/deleteAfterDays/1",
			SaveAs:   SaveAs{SaveBucket: "test", SaveKey: "test", DeleteAfterDays: &expiredDays},
		},
	}

	for _, testCase := range testCases {
		fopString, err := testCase.SaveAs.ToString()
		if err != nil {
			t.Errorf("SaveAs to string error: %v", err)
		}

		if fopString != testCase.Expected {
			t.Errorf("SaveAs to string error: %s, expected: %s", fopString, testCase.Expected)
		}

		t.Logf("SaveAs to string: %s", fopString)
	}

}

func TestMkZipArgsToString(t *testing.T) {
	testCases := []struct {
		Name     string
		Expected string
		MkZip    MkZipArgs
	}{
		{
			Name:     "test_mkzip_with_url",
			Expected: "mkzip/2/url/aHR0cDovL2V4YW1wbGUuY29tL2ZpbGUudHh0/alias/ZmlsZS50eHQ=",
			MkZip:    MkZipArgs{URLsMap: map[string]string{"http://example.com/file.txt": "file.txt"}},
		},
		{
			Name:     "test_mkzip_with_url_and_alias",
			Expected: "mkzip/2/url/aHR0cDovL2V4YW1wbGUuY29tL2ZpbGUudHh0/alias/ZmlsZS50eHQ=/url/aHR0cDovL2V4YW1wbGUuY29tL2ZpbGUyLnR4dA==/alias/ZmlsZTIudHh0",
			MkZip:    MkZipArgs{URLsMap: map[string]string{"http://example.com/file.txt": "file.txt", "http://example.com/file2.txt": "file2.txt"}},
		},
		{
			Name:     "test_mkzip_with_url_and_alias_and_encoding",
			Expected: "mkzip/2/encoding/YWJj/url/aHR0cDovL2V4YW1wbGUuY29tL2ZpbGUudHh0/alias/ZmlsZS50eHQ=/url/aHR0cDovL2V4YW1wbGUuY29tL2ZpbGUyLnR4dA==/alias/ZmlsZTIudHh0",
			MkZip:    MkZipArgs{URLsMap: map[string]string{"http://example.com/file.txt": "file.txt", "http://example.com/file2.txt": "file2.txt"}, Encoding: "abc"},
		},
	}

	for _, testCase := range testCases {
		fopString, err := testCase.MkZip.ToString()
		if err != nil {
			t.Errorf("%s MkZipArgs to string error: %v", testCase.Name, err)
		}

		if fopString != testCase.Expected {
			t.Errorf("%s MkZipArgs to string error: %s, expected: %s", testCase.Name, fopString, testCase.Expected)
		}

		t.Logf("%s MkZipArgs to string: %s", testCase.Name, fopString)
	}
}
