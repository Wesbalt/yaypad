package main

// https://docs.microsoft.com/en-us/windows/win32/winprog/windows-data-types
type (
	BOOL       uint32
	BOOLEAN    byte
	BYTE       byte
	SHORT      int16
	WORD       uint16
	DWORD      uint32
	DWORD64    uint64
	HANDLE     uintptr
	HLOCAL     uintptr
	LONG       int32
	LPVOID     uintptr
	SIZE_T     uintptr
	INT        int32
	UINT       uint32
	ULONG_PTR  uintptr
	ULONGLONG  uint64
)
