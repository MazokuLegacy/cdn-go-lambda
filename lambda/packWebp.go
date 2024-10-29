package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

func packWebp(inputs [][]byte, _ int) ([]byte, error) {
	//preparing inputs
	dir := "/tmp/pack"
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return nil, err
	}
	err = processFilesParallel(inputs)
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	//preparing output
	outPath := "/tmp/pack/output.webp"
	outFile, err := os.Create(outPath)
	if err != nil {
		return nil, err
	}
	defer outFile.Close()

	cmd := exec.Command("magick",
		"convert", "+append",
		"/tmp/pack/card*.png",
		"-resize", "200x",
		outPath)

	err = cmd.Start()
	if err != nil {
		fmt.Println("Error starting command:", err)
		return nil, err
	}

	err = cmd.Wait()
	if err != nil {
		fmt.Println("Error waiting for command:", err)
		return nil, err
	}

	output, err := io.ReadAll(outFile)
	return output, nil
}
