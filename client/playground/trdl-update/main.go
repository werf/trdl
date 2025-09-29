package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func main() {
	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	}

	fmt.Println("running exe:", exePath)

	go func() {
		for i := 0; i < 30; i++ {
			exe, err := os.Executable()
			if err != nil {
				fmt.Printf("[exe %02d] os.Executable error: %v\n", i, err)
				time.Sleep(150 * time.Millisecond)
				continue
			}

			res, err := filepath.EvalSymlinks(exe)
			if err != nil {
				fmt.Printf("[exe %02d] EvalSymlinks error: %v\n", i, err)
			} else {
				fmt.Printf("[exe %02d] exe=%s -> %s\n", i, exe, res)
			}

			time.Sleep(150 * time.Millisecond)
		}
	}()

	time.Sleep(500 * time.Millisecond)

	go func() {
		oldPath := exePath + ".old"

		fmt.Println("[updater] renaming exe -> .old")
		if err := os.Rename(exePath, oldPath); err != nil {
			fmt.Println("[updater] rename error:", err)
			return
		}

		time.Sleep(500 * time.Millisecond)

		fmt.Println("[updater] removing .old exe")
		if err := os.Remove(oldPath); err != nil {
			fmt.Println("[updater] remove error:", err)
		}
	}()

	time.Sleep(6 * time.Second)

	fmt.Println("done")
}
