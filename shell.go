package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Оболочка командной строки
func shell(fs *FileSystem) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("$ ")
		scanner.Scan()
		read := scanner.Text()
		if read == "quit" || read == "exit" {
			break
		}

		readWithArgs := strings.Split(read, " ")

		if len(readWithArgs) <= 0 {
			continue
		}

		if readWithArgs[0] == "dir" || readWithArgs[0] == "ls" {
			folder := fs.curFolder

			if len(readWithArgs) > 1 {
				arg, isAbsolute := ParsePath(readWithArgs[1])

				if arg[0] == ".." && folder.Parent != nil {
					folder = folder.Parent

				} else {
					var start *File

					if isAbsolute {
						start = &fs.rootFolder
					} else {
						start = fs.curFolder
					}

					path := FindParentTo(start, arg)
					if path != nil && path.FileSize == 0 {
						folder = path
					}

					if len(arg) == 0 && readWithArgs[1][0] == '/' {
						folder = &fs.rootFolder
					}

				}
			}

			fmt.Printf("Папка: %s\n", StringifyPath(folder, ""))
			fmt.Println(".\n..")
			for _, v := range folder.DirNode {
				if !v.Hidden {
					fmt.Println(v.Name)
				}
			}
		}

		switch readWithArgs[0] {
		case "md", "mkdir":
			arg := readWithArgs[1]
			fs.CreateFolder(arg)

		case "mf", "mkfile":
			arg := readWithArgs[1]
			fs.CreateFile(arg)

		case "cd":
			arg, isAbsolute := ParsePath(readWithArgs[1])

			if arg[0] == ".." && fs.curFolder.Parent != nil {
				fs.curFolder = fs.curFolder.Parent
				break
			}

			var start *File

			if isAbsolute {
				start = &fs.rootFolder
			} else {
				start = fs.curFolder
			}

			path := FindParentTo(start, arg)
			if path != nil && path.FileSize == 0 {
				fs.curFolder = path
			}

			if len(arg) == 0 && readWithArgs[1][0] == '/' {
				fs.curFolder = &fs.rootFolder
			}

		case "rm":
			arg, isAbsolute := ParsePath(readWithArgs[1])

			var start *File
			if isAbsolute {
				start = &fs.rootFolder
			} else {
				start = fs.curFolder
			}

			file := FindParentTo(start, arg)
			if file != nil {
				if file.FileSize == 0 {
					out := fs.DeleteFolder(file)

					if out == 1 {
						if yesNoDialog("Папка не пуста, удалить рекурсивно?") {
							// Рекурсивное удаление папки
							if r := fs.DeleteFolderRecoursively(file); r != 0 {
								fmt.Println("Невозможно удалить каталог!")
							}
						}
					} else {
						fmt.Println("Невозможно удалить каталог!")
					}
				} else {
					if fs.DeleteFile(file) != 0 {
						fmt.Println("Невозможно удалить файл!")
					}
				}
			}

		case "cat":
			arg, isAbsolute := ParsePath(readWithArgs[1])

			var start *File
			if isAbsolute {
				start = &fs.rootFolder
			} else {
				start = fs.curFolder
			}

			file := FindParentTo(start, arg)
			if file == nil && readWithArgs[1][0] == '/' {
				file = &fs.rootFolder
			}
			if file != nil {
				if file.FileSize == 0 {
					fmt.Printf("Папка %s, создана %v, изменена %v, файлов: %d\n\n", file.Name, file.CreationTime, file.Time, len(file.DirNode))
				} else {
					fmt.Printf("Файл %s, создан %v, изменен %v, размер: %d\n", file.Name, file.CreationTime, file.Time, file.FileSize)

					bytesToRead := int(file.FileSize)
					if len(readWithArgs) > 2 {
						bytesToRead, _ = strconv.Atoi(readWithArgs[2])
					}

					fs.ReadFile(file, bytesToRead)
				}
			}

		case "rename":
			arg, isAbsolute := ParsePath(readWithArgs[1])

			var start *File
			if isAbsolute {
				start = &fs.rootFolder
			} else {
				start = fs.curFolder
			}

			file := FindParentTo(start, arg)
			if file != nil {
				fs.Rename(file, readWithArgs[2])
			}

		case "echop":
			if len(readWithArgs) < 3 {
				break
			}

			arg, isAbsolute := ParsePath(readWithArgs[2])

			var start *File
			if isAbsolute {
				start = &fs.rootFolder
			} else {
				start = fs.curFolder
			}

			file := FindParentTo(start, arg)
			if file == nil {
				file = fs.CreateFile(readWithArgs[2])
			}

			echoLen := len(readWithArgs[1])
			if fs.WriteFile(file, echoLen) == 2 {
				fmt.Println("Невозможно записать в файл: операция запрещена")
			}

		case "chattr":
			if len(readWithArgs) < 3 {
				break
			}

			arg, isAbsolute := ParsePath(readWithArgs[2])

			var start *File
			if isAbsolute {
				start = &fs.rootFolder
			} else {
				start = fs.curFolder
			}

			file := FindParentTo(start, arg)
			if file != nil {
				readonly, hidden, system, archive := fs.GetAttributes(file)

				switch readWithArgs[1] {
				case "+r":
					readonly = true
				case "+h":
					hidden = true
				case "+s":
					system = true
				case "+a":
					archive = true
				case "-r":
					readonly = false
				case "-h":
					hidden = false
				case "-s":
					system = false
				case "-a":
					archive = false
				}

				fs.SetAttributes(file, readonly, hidden, system, archive)
			}
		}

		fmt.Println()
	}
}

func yesNoDialog(question string) bool {
	fmt.Printf("%s (y/n): ", question)
	s := bufio.NewScanner(os.Stdin)
	var str string
	for {
		s.Scan()
		str = s.Text()
		if str == "y" || str == "n" {
			break
		}
	}

	if str == "y" {
		return true
	}

	return false
}
