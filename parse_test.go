package sf

import "fmt"

func ExampleParseDict() {
	d, err := ParseDict([]string{`foo=1`, `bar=2`})
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(d.Encode())
	}

	// Output:
	// foo=1, bar=2
}

func ExampleParseDictLine() {
	tests := []string{
		`en="Applepie", da=:w4ZibGV0w6ZydGU=:`,
		`a=?0, b, c; foo=bar`,
		`rating=1.5, feelings=(joy sadness)`,
		`a=(1 2), b=3, c=4;aa=bb, d=(5 6);valid`,
	}
	for _, t := range tests {
		d, err := ParseDictLine(t)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(d.Encode())
		}
	}

	// Output:
	// en="Applepie", da=:w4ZibGV0w6ZydGU=:
	// a=?0, b, c;foo=bar
	// rating=1.5, feelings=(joy sadness)
	// a=(1 2), b=3, c=4;aa=bb, d=(5 6);valid
}

func ExampleParseList() {
	d, err := ParseList([]string{`sugar, tea`, `rum`})
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(d.Encode())
	}

	// Output:
	// sugar, tea, rum
}

func ExampleParseListLine() {
	tests := []string{
		`sugar, tea, rum`,
	}
	for _, t := range tests {
		d, err := ParseListLine(t)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(d.Encode())
		}
	}

	// Output:
	// sugar, tea, rum
}

func ExampleParseItemLine() {
	tests := []string{
		`5; foo=bar`,
		`4.5`,
		`"hello world"`,
		`foo123/456`,
	}
	for _, t := range tests {
		d, err := ParseItemLine(t)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(d.Encode())
		}
	}

	// Output:
	// 5;foo=bar
	// 4.5
	// "hello world"
	// foo123/456
}
