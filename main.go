package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func signalNotify(f func(s os.Signal), sig ...os.Signal) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, sig...)
	go func() {
		s := <-ch
		f(s)
	}()
}

func mountAndWait(username, password, mountpoint string) error {
	done := make(chan struct{})

	root := NewRoot(username, password)

	signalNotify(func(s os.Signal) {
		err := root.Unmount()
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to unmount:", err)
		}
		done <- struct{}{}
	}, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	err := root.Mount(mountpoint)
	if err != nil {
		return err
	}
	<-done

	return nil
}

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

	err := mountAndWait(*username, *password, args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to mount:", err)
	}

	return 0
}

func main() {
	os.Exit(run())
}
