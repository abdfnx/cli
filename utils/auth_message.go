package utils

import (
	"os"
	"fmt"
)

func AuthMessage() {
	fmt.Println("You're not authenticated, to authenticate run `secman auth login`")

	os.Exit(0)
}
