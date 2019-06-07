package main

import "time"

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
