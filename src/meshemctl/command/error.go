package command

import (
	"fmt"
	"os"
)

func ExitWithError(err error) {
	fmt.Fprintln(os.Stderr, "Error: ", err)
	os.Exit(1)
}
