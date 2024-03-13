package cc

import (
	"github.com/kelindar/gocc/internal/asm"
	"github.com/kelindar/gocc/internal/config"
	"github.com/kelindar/gocc/internal/golang/objfile"
)

type Disassembler struct {
	arch         *config.Arch
	disassembler []string
}

func NewDisassembler(arch *config.Arch) (*Disassembler, error) {
	var (
		dasm    []string
		objdump string
		err     error
	)

	objdump, err = config.FindClangObjdump()
	if err != nil {
		return nil, err
	}
	dasm = append(dasm, objdump)

	return &Disassembler{
		arch:         arch,
		disassembler: dasm,
	}, nil
}

// Disassemble disassembles the object file
func (d *Disassembler) Disassemble(assemblyPath, objectPath string) ([]asm.Function, error) {
	// Parse the assembly file
	assembly, err := asm.ParseAssembly(d.arch, assemblyPath)
	if err != nil {
		return nil, err
	}

	objF, err := objfile.Open(objectPath)
	if err != nil {
		return nil, err
	}
	disasm, err := objF.Disasm()
	if err != nil {
		return nil, err
	}

	disassembler := d.disassembler
	if d.arch.Disassembler != nil {
		disassembler = append(disassembler, d.arch.Disassembler...)
	}
	disassembler = append(disassembler, "-d", objectPath)

	// Run the disassembler
	dump, err := runCommand(disassembler[0], disassembler[1:]...)
	if err != nil {
		return nil, err
	}

	// Parse the object dump and map machine code to assembly
	err = asm.ParseClangObjectDump(d.arch, dump, assembly, disasm)
	if err != nil {
		return nil, err
	}

	return assembly, nil
}
