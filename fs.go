package main

import (
	"container/list"
	"fmt"
	"strings"
	"time"
)

// MaxDiskSize определяет максимальный размер диска в 8-байтовых блоках
// 16777216 = 128 8-байтных чисел (1024 байта, 1 кбайт) * 1024 * 128 = 128 мб
const MaxDiskSize = 16777216

// MaxBlockCount показывает максимальное количество блоков размером в 1 килобайт
// 131072 = 1 * 1024 * 128 = 128 мб
const MaxBlockCount = 131072

// FAT - file allocation table, хранит адреса кластеров
type FAT struct {
	Blocks [MaxBlockCount]int
}

// BlockManager представляет менеджер свободных дисковых секторов
type BlockManager struct {
	blockList *list.List
}

// File представляет запись файла в памяти, а также часть блока управления файлом в памяти
type File struct {
	Name string
	// Readonly, Archive, System, Hidden
	Readonly          bool
	Archive           bool
	System            bool
	Hidden            bool
	CreationTime      time.Time
	Time              time.Time
	FirstBlockAddress int32
	FileSize          uint32
	Current           *File
	Parent            *File
	DirNode           []*File
	filePosition      int
}

// FileSystem это представление диска вместе с несколькими уровнями абстракци
type FileSystem struct {
	blockManager *BlockManager
	allocTable   FAT
	rootFolder   File
	curFolder    *File
}

// NewBlockManager создает новый менеджер свободных блоков
func NewBlockManager() *BlockManager {
	blockList := list.New()
	for i := 0; i < MaxBlockCount; i++ {
		blockList.PushBack(int32(i))
	}
	return &BlockManager{blockList}
}

// GetFreeBlock пытается получить доступ к свободному блоку,
// если получается: возвращает адрес, иначе: -1
func (bm *BlockManager) GetFreeBlock() int32 {
	e := bm.blockList.Front()
	if e == nil {
		return -1
	}

	value := e.Value.(int32)
	bm.blockList.Remove(e)

	return value
}

// AddBlock помечает указанный блок свободным
func (bm *BlockManager) AddBlock(blockAddress int) {
	bm.blockList.PushFront(blockAddress)
}

// CreateFile создает файл в указанном месте
func (fs *FileSystem) CreateFile(name string) *File {
	path, isAbsolute := ParsePath(name)

	var fileName string
	if len(path) > 0 {
		fileName = path[len(path)-1]
	}

	file := &File{Name: fileName, CreationTime: time.Now(), Time: time.Now(), FileSize: 0}

	path = path[:len(path)-1]
	if isAbsolute {
		file.Parent = FindParentTo(&fs.rootFolder, path)
	} else {
		file.Parent = FindParentTo(fs.curFolder, path)
	}

	if file.Parent.FileSize != 0 {
		return nil
	}

	file.Parent.DirNode = append(file.Parent.DirNode, file)

	block := fs.blockManager.GetFreeBlock()

	if block == -1 {
		return nil
	}

	fs.allocTable.Blocks[block] = -1

	file.FirstBlockAddress = block

	file.FileSize = 1

	return file
}

// DeleteFile удаляет файл, освобождает дисковое пространство и сектора памяти
func (fs *FileSystem) DeleteFile(file *File) int {
	for blockAddress := file.FirstBlockAddress; blockAddress != -1; blockAddress = int32(fs.allocTable.Blocks[blockAddress]) {
		fs.blockManager.AddBlock(int(blockAddress))
	}

	for i, v := range file.Parent.DirNode {
		if v == file {
			file.Parent.DirNode = append(file.Parent.DirNode[:i], file.Parent.DirNode[i+1:]...)
		}
	}

	return 0
}

// OpenFile открывает файл для редактирования
func (fs *FileSystem) OpenFile(name string) *File {
	path, isAbsolute := ParsePath(name)

	if isAbsolute {
		return FindParentTo(&fs.rootFolder, path)
	}

	return FindParentTo(fs.curFolder, path)
}

// CloseFile сбрасывает служебную информацию о редактировании процесса
func (fs *FileSystem) CloseFile(file *File) {
	// Close file
	file.filePosition = 0
}

// ReadFile "читает" size байтов из файла из служебной позиции
func (fs *FileSystem) ReadFile(file *File, size int) {
	// TBD
	for i, blockCounter, curBlock := file.filePosition, file.filePosition%1024, file.FirstBlockAddress; i < file.filePosition+size; i++ {
		if i%1024 == 0 && i != 0 {
			fmt.Printf("Закончен счет блока %d в позиции %d файла %s, начат следующий блок\n", blockCounter, curBlock, file.Name)
			blockCounter++
			curBlock = int32(fs.allocTable.Blocks[blockCounter])
			if curBlock == -1 {
				fmt.Println("Ошибка чтения файла!")
				break
			}

		}
		if i >= int(file.FileSize) {
			break
		}
	}

	fmt.Println("Чтение файла закончено")
}

// WriteFile пишет size байтов в файл и если это возможно, увеличивает размер файла
func (fs *FileSystem) WriteFile(file *File, size int) int {
	if file.FileSize == 0 {
		return 1
	}

	for i, blockCounter, curBlock := file.filePosition, file.filePosition%1024, file.FirstBlockAddress; i < file.filePosition+size; i++ {
		if i%1024 == 0 && i != 0 {
			fmt.Printf("Закончена запись блока %d в позиции %d файла %s, начат следующий блок\n", blockCounter, curBlock, file.Name)
			blockCounter++
			curBlock = int32(fs.allocTable.Blocks[blockCounter])
			if curBlock == -1 {
				if requiredBlock := fs.blockManager.GetFreeBlock(); requiredBlock != -1 {
					fmt.Printf("%d-й блок %d успешно начат!\n", blockCounter, requiredBlock)
					fs.allocTable.Blocks[blockCounter] = int(requiredBlock)
				} else {
					fmt.Println("Ошибка памяти!")
					return 2
				}
			}

		}
		if i >= int(file.FileSize) {
			file.FileSize++
		}
	}

	fmt.Println("Запись в файл закончена")

	return 0
}

// Seek позиционирует файл
func (fs *FileSystem) Seek(file *File, position int) {
	// TBD
	file.filePosition = position
}

// GetAttributes возвращает 4 атрибута файла
func (fs *FileSystem) GetAttributes(file *File) (bool, bool, bool, bool) {
	return file.Readonly, file.Hidden, file.System, file.Archive
}

// SetAttributes устанавливает атрибуты файла, если это возможно
func (fs *FileSystem) SetAttributes(file *File, readonly, archive, system, hidden bool) {
	if file.Readonly || file.System {
		return
	}

	file.Readonly = readonly
	file.Hidden = hidden
	file.System = system
	file.Archive = archive
}

// Rename переименовывает файл, если это возможно
func (fs *FileSystem) Rename(file *File, name string) {
	if file.Readonly || file.System {
		return
	}

	file.Name = name
}

// CreateFolder создает каталог в указанном месте
func (fs *FileSystem) CreateFolder(name string) *File {
	path, isAbsolute := ParsePath(name)

	var fileName string
	if len(path) > 0 {
		fileName = path[len(path)-1]
	}

	folder := &File{Name: fileName, CreationTime: time.Now(), Time: time.Now(), FileSize: 0}
	folder.Current = folder

	path = path[:len(path)-1]

	if isAbsolute {
		folder.Parent = FindParentTo(&fs.rootFolder, path)
	} else {
		folder.Parent = FindParentTo(fs.curFolder, path)
	}

	if folder.Parent.FileSize != 0 {
		return nil
	}

	folder.Parent.DirNode = append(folder.Parent.DirNode, folder)

	return folder
}

// DeleteFolder удаляет пустой каталог
func (fs *FileSystem) DeleteFolder(file *File) int {
	/*if file.FileSize != 0 {
		return -1
	}

	// Встроенное рекурсивное удаление
	for _, v := range file.DirNode {
		if v.FileSize == 0 {
			fs.DeleteFolder(v)
		}
	}

	fs.CloseFolder(file)*/

	//

	if len(file.DirNode) > 0 {
		return 1
	}

	if file.Readonly || file.System {
		return 2
	}

	if file.Parent == file || file.Parent == nil {
		return 3
	}

	fs.CloseFolder(file)

	for i, v := range file.Parent.DirNode {
		if v == file {
			file.Parent.DirNode = append(file.Parent.DirNode[:i], file.Parent.DirNode[i+1:]...)
		}
	}

	return 0
}

// DeleteFolderRecoursively осуществляет рекурсивное удаление файлов
func (fs *FileSystem) DeleteFolderRecoursively(file *File) int {
	for _, v := range file.DirNode {
		if v.FileSize == 0 {
			if len(v.DirNode) == 0 {
				if r := fs.DeleteFolder(v); r != 0 {
					return r
				}
			} else {
				if r := fs.DeleteFolderRecoursively(v); r != 0 {
					return r
				}

				if r := fs.DeleteFolder(v); r != 0 {
					return r
				}
			}
		} else {
			if r := fs.DeleteFile(v); r != 0 {
				return r
			}
		}
	}

	return fs.DeleteFolder(file)
}

// OpenFolder открывает папку для редактирования
func (fs *FileSystem) OpenFolder(name string) *File {
	path, isAbsolute := ParsePath(name)

	if isAbsolute {
		return FindParentTo(&fs.rootFolder, path)
	}

	return FindParentTo(fs.curFolder, path)
}

// CloseFolder сбрасывает служебную информацию
func (fs *FileSystem) CloseFolder(file *File) {
	// Folder is closed
}

// NewFileSystem создает экземпляр файловой системы
func NewFileSystem() *FileSystem {
	fs := &FileSystem{blockManager: NewBlockManager()}
	// Установка корневой папки
	fs.rootFolder = File{Name: "root", CreationTime: time.Now(), Time: time.Now(), FileSize: 0}
	fs.rootFolder.Current = &fs.rootFolder
	fs.rootFolder.Parent = &fs.rootFolder
	fs.curFolder = &fs.rootFolder

	return fs
}

// ParsePath представляет путь в виде списка и указывает, абсолютный он или относительный
func ParsePath(path string) ([]string, bool) {
	absolute := false
	if len(path) > 0 && path[0] == '/' {
		absolute = true
		path = path[1:]
	}

	elements := strings.Split(path, "/")

	return elements, absolute
}

// FindParentTo рекурсивно спускается в каталоги и ищет там экземпляр файла
func FindParentTo(current *File, path []string) *File {
	if current.FileSize != 0 && len(path) != 0 {
		return nil
	}

	if len(path) == 0 {
		return current
	}

	cur := path[0]

	for _, v := range current.DirNode {
		if len(path) == 1 {
			if v.Name == cur {
				return v
			}
		} else {
			if v.Name == cur {
				return FindParentTo(v, path[1:])
			}
		}
	}

	return nil
}

// StringifyPath выводит расположение файла в строку
func StringifyPath(file *File, rest string) string {
	if file.Name == "root" {
		if rest == "" {
			return "/"
		}

		return rest
	}

	return StringifyPath(file.Parent, "/"+file.Name+""+rest)
}
