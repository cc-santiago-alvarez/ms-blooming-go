package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Tipos de línea del respaldo.
const (
	lineRun    = "corrida"
	lineChange = "cambio"
)

// backupHeader es la primera línea del respaldo: de qué corrida y de qué base son los cambios. Nunca lleva la
// URL.
type backupHeader struct {
	Kind       string    `bson:"kind"`
	Run        time.Time `bson:"run"`
	Env        string    `bson:"env"`
	DB         string    `bson:"db"`
	ReplicaSet string    `bson:"replicaSet"`
	Only       []string  `bson:"only"`
	Revision   string    `bson:"revision,omitempty"`
	Modified   *bool     `bson:"modified,omitempty"`
}

// backupChange es un cambio de un documento: los valores de antes y de después de los campos que toca, con
// updatedAt, y el historial que registra. Absent son los campos que no estaban antes.
type backupChange struct {
	Kind       string        `bson:"kind"`
	Fix        string        `bson:"fix"`
	Collection string        `bson:"collection"`
	ID         bson.RawValue `bson:"_id"`
	Before     bson.D        `bson:"before"`
	Absent     []string      `bson:"absent,omitempty"`
	After      bson.D        `bson:"after"`
	HistoryID  string        `bson:"historyId,omitempty"`
}

// withUpdatedAt agrega updatedAt al cambio: el valor anterior, o la marca de que no estaba, y el nuevo.
func (c *backupChange) withUpdatedAt(old bson.RawValue, runAt time.Time) {
	if old.Type == 0 {
		c.Absent = append(c.Absent, "updatedAt")
	} else {
		c.Before = append(c.Before, bson.E{Key: "updatedAt", Value: old})
	}
	c.After = append(c.After, bson.E{Key: "updatedAt", Value: runAt})
}

// backupWriter escribe el respaldo de una corrida de -apply, una línea en Extended JSON canónico por cambio,
// que conserva los tipos BSON. El archivo se crea con el primer cambio, nunca pisa uno existente y solo lo
// puede leer su dueño.
type backupWriter struct {
	path    string
	header  backupHeader
	file    *os.File
	changes int
}

// newBackupWriter prepara el respaldo; todavía no crea el archivo.
func newBackupWriter(path string, header backupHeader) *backupWriter {
	return &backupWriter{path: path, header: header}
}

// Write guarda el cambio y lo baja a disco, antes de que se escriba en la base.
func (w *backupWriter) Write(change backupChange) error {
	if w.file == nil {
		if err := os.MkdirAll(filepath.Dir(w.path), 0o700); err != nil {
			return err
		}
		f, err := os.OpenFile(w.path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return err
		}
		w.file = f
		if err := w.writeLine(w.header); err != nil {
			return err
		}
	}
	if err := w.writeLine(change); err != nil {
		return err
	}
	w.changes++
	return nil
}

// writeLine escribe v como una línea y la baja a disco. Si no puede, deja el archivo como estaba, sin una
// línea a medias.
func (w *backupWriter) writeLine(v any) error {
	data, err := bson.MarshalExtJSON(v, true, false)
	if err != nil {
		return err
	}
	offset, err := w.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	_, err = w.file.Write(append(data, '\n'))
	if err == nil {
		err = w.file.Sync()
	}
	if err != nil {
		_ = w.file.Truncate(offset)
		_, _ = w.file.Seek(offset, io.SeekStart)
	}
	return err
}

// Close cierra el archivo, si se creó.
func (w *backupWriter) Close() error {
	if w.file == nil {
		return nil
	}
	return w.file.Close()
}

// backupLine es una línea leída del respaldo: la cabecera o un cambio.
type backupLine struct {
	Kind       string        `bson:"kind"`
	Run        time.Time     `bson:"run"`
	Env        string        `bson:"env"`
	DB         string        `bson:"db"`
	Fix        string        `bson:"fix"`
	Collection string        `bson:"collection"`
	ID         bson.RawValue `bson:"_id"`
	Before     bson.Raw      `bson:"before"`
	Absent     []string      `bson:"absent"`
	After      bson.Raw      `bson:"after"`
	HistoryID  string        `bson:"historyId"`
}

// readBackup lee la cabecera y los cambios de un respaldo, en el orden en que se aplicaron. Una última línea
// sin salto de línea es una escritura a medias, cuyo cambio no llegó a la base: se ignora y torn lo avisa.
func readBackup(path string) (header backupLine, changes []backupLine, torn bool, err error) {
	f, err := os.Open(path)
	if err != nil {
		return backupLine{}, nil, false, err
	}
	defer func() { _ = f.Close() }()

	changes = []backupLine{}
	reader := bufio.NewReader(f)
	for n := 1; ; n++ {
		data, readErr := reader.ReadBytes('\n')
		if readErr != nil && readErr != io.EOF {
			return backupLine{}, nil, false, readErr
		}
		if readErr == io.EOF {
			torn = len(data) > 0
			break
		}
		var line backupLine
		if err := bson.UnmarshalExtJSON(data, true, &line); err != nil {
			return backupLine{}, nil, false, fmt.Errorf("línea %d del respaldo: %w", n, err)
		}
		switch {
		case n == 1 && line.Kind == lineRun:
			header = line
		case n > 1 && line.Kind == lineChange:
			changes = append(changes, line)
		default:
			return backupLine{}, nil, false, fmt.Errorf("línea %d del respaldo: tipo %q inesperado", n, line.Kind)
		}
	}
	if header.Kind != lineRun {
		return backupLine{}, nil, false, errors.New("el respaldo está vacío o no empieza con la cabecera de la corrida")
	}
	return header, changes, torn, nil
}

// build devuelve el commit con el que se compiló la herramienta y si tenía cambios sin commitear, si se
// conocen.
func build() (revision string, modified *bool) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", nil
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			m := s.Value == "true"
			modified = &m
		}
	}
	return revision, modified
}

// buildText muestra el commit de la herramienta.
func buildText(revision string, modified *bool) string {
	switch {
	case revision == "":
		return "commit desconocido"
	case modified != nil && *modified:
		return "commit " + revision + " con cambios sin commitear"
	}
	return "commit " + revision
}
