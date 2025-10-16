package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func main() {
	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	}

	fmt.Println("running exe:", exePath)

	getRealExecutable := func() string {
		exe, err := os.Executable()
		if err != nil {
			fmt.Println("[getRealExecutable] os.Executable error:", err)
			return exePath
		}

		res, err := filepath.EvalSymlinks(exe)
		if err != nil {
			fmt.Printf("[getRealExecutable] EvalSymlinks error: %v, using exe path instead\n", err)
			return exe
		}

		return res
	}

	go func() {
		for i := 0; i < 30; i++ {
			res := getRealExecutable()
			fmt.Printf("[exe %02d] -> %s\n", i, res)
			time.Sleep(150 * time.Millisecond)
		}
	}()

	time.Sleep(500 * time.Millisecond)

	go func() {
		dir := filepath.Dir(exePath)
		newPath := filepath.Join(dir, ".new")
		oldPath := exePath + ".old"

		if err := copyFile(exePath, newPath); err != nil {
			fmt.Println("[updater] copy .new error:", err)
			return
		}
		fmt.Println("[updater] created .new copy")

		fmt.Println("[updater] renaming exe -> .old")
		if err := os.Rename(exePath, oldPath); err != nil {
			fmt.Println("[updater] rename error:", err)
			return
		}

		time.Sleep(500 * time.Millisecond)

		fmt.Println("[updater] moving .new -> exe")
		if err := os.Rename(newPath, exePath); err != nil {
			fmt.Println("[updater] move .new -> exe error:", err)
			return
		}

		fmt.Println("[updater] removing .old exe")
		if err := os.Remove(oldPath); err != nil {
			fmt.Println("[updater] remove .old error:", err)
		}
	}()

	time.Sleep(6 * time.Second)
	fmt.Println("done")
}
