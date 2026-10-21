package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/retro-ware/gameimport"
	"github.com/retro-ware/wiz6-go/internal/dbs"
	"github.com/retro-ware/wiz6-go/internal/drv"
)

// dataFile is whether the game reads a file of this name (the EGA set; CGA, Tandy and
// Hercules files are left out).
func dataFile(name string) bool {
	switch strings.ToUpper(path.Ext(name)) {
	case ".DBS", ".HDR", ".EGA", ".PIC", ".SND":
		return true
	}
	return false
}

// importGame finds Wizardry VI in src (folders, zips, CD images, GOG installers, loose
// files), copies the data files from the folder SCENARIO.DBS is in to dir, and writes
// COVER.PNG for goplay's disk label from the title page.
func importGame(src, dir string) error {
	files, err := gameimport.Find(src, func(name string) bool { return dataFile(name) })
	if err != nil {
		return err
	}
	home := ""
	for name := range files {
		if strings.EqualFold(path.Base(name), "SCENARIO.DBS") && (home == "" || len(name) < len(home)) {
			home = path.Dir(name)
		}
	}
	if home == "" {
		return fmt.Errorf("no Wizardry VI (SCENARIO.DBS) in %s", src)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for name, b := range files {
		if path.Dir(name) != home {
			continue
		}
		if err := os.WriteFile(filepath.Join(dir, strings.ToUpper(path.Base(name))), b, 0o644); err != nil {
			return err
		}
	}
	return writeCover(dir)
}

// writeCover writes TITLEPAG.EGA as dir/COVER.PNG.
func writeCover(dir string) error {
	px, err := dbs.LoadScreen(filepath.Join(dir, "TITLEPAG.EGA"))
	if err != nil {
		return err
	}
	pal := make(color.Palette, 16)
	for i, c := range drv.Palette {
		pal[i] = color.RGBA{c[0], c[1], c[2], 255}
	}
	m := image.NewPaletted(image.Rect(0, 0, drv.ScreenWidth, drv.ScreenHeight), pal)
	copy(m.Pix, px[:])
	f, err := os.Create(filepath.Join(dir, "COVER.PNG"))
	if err != nil {
		return err
	}
	if err := png.Encode(f, m); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
