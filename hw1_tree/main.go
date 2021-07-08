package main

import (
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
)

type Node struct {
	fileInfo           fs.FileInfo
	isFolder           bool
	isLast             bool
	parentsIsLastFlags []bool
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}

func dirTree(out io.ReadWriter, path string, printFiles bool) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	fileInfo, _ := file.Stat()
	main_node := new(Node)
	main_node.isLast = false
	main_node.isFolder = true
	main_node.fileInfo = fileInfo

	err = build_tree(out, main_node, path, []bool{}, printFiles, true)
	if err != nil {
		return err
	}

	return nil
}

func build_tree(out io.ReadWriter, node *Node, path string, parentsIsLastFlags []bool, printFiles bool, isFirst bool) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	count_files := len(files)
	if !printFiles {
		for _, fl := range files {
			if !fl.IsDir() {
				count_files--
			}
		}
	}

	if count_files == 0 {
		return nil
	}

	sort.SliceStable(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	num_file := 1
	for _, fl := range files {
		if !printFiles {
			if !fl.IsDir() {
				continue
			}
		}

		ch_node := new(Node)
		ch_node.fileInfo = fl
		ch_node.isLast = count_files == 1 || num_file == count_files
		ch_node.parentsIsLastFlags = parentsIsLastFlags
		ch_node.parentsIsLastFlags = append(ch_node.parentsIsLastFlags, ch_node.isLast)

		if printFiles || ch_node.fileInfo.IsDir() {
			out.Write([]byte(getFileRow(ch_node, ch_node.parentsIsLastFlags)))
		}

		out.Write([]byte("\n"))
		num_file++

		if ch_node.fileInfo.IsDir() {
			build_tree(out, ch_node, path+"/"+ch_node.fileInfo.Name(), ch_node.parentsIsLastFlags, printFiles, false)
		}

	}

	return nil
}

func getFileRow(node *Node, parentsIsLastFlags []bool) string {
	res := ""
	indent := ""
	countParents := len(parentsIsLastFlags)

	for i, fl := range parentsIsLastFlags {
		if countParents == i+1 {
			if node.isLast {
				indent += "└───"
			} else {
				indent += "├───"
			}
		} else {
			if fl == true {
				indent = indent + "	"
			} else {
				indent = indent + "│	"
			}
		}
	}

	res = indent + node.fileInfo.Name()
	if !node.fileInfo.IsDir() {
		res += " ("
		if node.fileInfo.Size() == 0 {
			res += "empty"
		} else {
			res += strconv.Itoa(int(node.fileInfo.Size()))
			res += "b"
		}

		res += ")"
	}
	return res
}
