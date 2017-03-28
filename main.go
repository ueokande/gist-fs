package main

import (
	"flag"
	"fmt"
	"os"
)

func run() int {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -u <username> -p <password> <directory>\n", os.Args[0])
	}

	username := flag.String("u", "", "`username` for authentication")
	password := flag.String("p", "", "`password` for authentication")
	flag.Parse()

	args := flag.Args()

	if len(os.Args) == 1 {
		flag.Usage()
		return 1
	}
	if *username == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "both -u and -p are required")
		return 1
	}
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "mountpoint is required")
		return 1
	}

	fmt.Println("username =", *username)
	fmt.Println("password =", *password)
	fmt.Println("mount to =", args[0])

	return 0
}

func main() {
	os.Exit(run())
}
