package utils

import (
	"os"
	"fmt"
)

func AuthMessage() {
	fmt.Println("You're not authenticated, to authenticate run `tran auth login`")

	os.Exit(0)
}
