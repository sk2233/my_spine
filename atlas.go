package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type AtlasItem struct {
	Name         string
	Rotate       bool
	X, Y         int
	W, H         int
	OrigW, OrigH int
	OrigX, OrigY int
	Index        int
}

type AtlasHeader struct {
	Image            string
	W, H             int
	Format           string
	WFilter, HFilter string
	Repeat           string
}

type Atlas struct {
	Header *AtlasHeader
	Items  []*AtlasItem
	Image  string
}

//
//func (a *Atlas) Save(path string) {
//	file, err := os.Open(BasePath + a.Image)
//	HandleErr(err)
//	img, err := png.Decode(file)
//	HandleErr(err)
//	file.Close()
//
//	for _, item := range a.Items {
//		temp := image.NewRGBA(image.Rect(0, 0, item.OrigW, item.OrigH))
//		draw.Draw(temp, image.Rect(item.OrigX, item.OrigY, item.OrigX+item.W, item.OrigY+item.H),
//			img, image.Pt(item.X, item.Y), draw.Over)
//		if item.Rotate {
//			temp = rotate90(temp)
//		}
//		file, err = os.Create(BasePath + path + "/" + item.Name + ".png")
//		HandleErr(err)
//		err = png.Encode(file, temp)
//		HandleErr(err)
//		file.Close()
//	}
//}

func ParseAtlas(path string) *Atlas {
	bs, err := os.ReadFile(BasePath + path)
	HandleErr(err)
	lines := strings.Split(string(bs), "\n")
	header := parseAtlasHeader(lines[1:6])
	items := parseAtlasItems(lines[6:])
	index := strings.LastIndex(path, "/")
	return &Atlas{
		Header: header,
		Items:  items,
		Image:  path[:index+1] + header.Image,
	}
}

func parseAtlasItems(items []string) []*AtlasItem {
	res := make([]*AtlasItem, 0)
	for i := 0; i+7 <= len(items); i += 7 {
		res = append(res, parseAtlasItem(items[i:i+7]))
	}
	return res
}

func parseAtlasItem(items []string) *AtlasItem {
	rotate := parseBoolList(items[1], "rotate")
	xy := parseIntList(items[2], "xy")
	size := parseIntList(items[3], "size")
	orig := parseIntList(items[4], "orig")
	if rotate[0] {
		size[0], size[1] = size[1], size[0]
		orig[0], orig[1] = orig[1], orig[0]
	}
	offset := parseIntList(items[5], "offset")
	index := parseIntList(items[6], "index")
	return &AtlasItem{
		Name:   strings.TrimSpace(items[0]),
		Rotate: rotate[0],
		X:      xy[0],
		Y:      xy[1],
		W:      size[0],
		H:      size[1],
		OrigW:  orig[0],
		OrigH:  orig[1],
		OrigX:  offset[0],
		OrigY:  offset[1],
		Index:  index[0],
	}
}

func parseAtlasHeader(items []string) *AtlasHeader {
	size := parseIntList(items[1], "size")
	format := parseStrList(items[2], "format")
	filter := parseStrList(items[3], "filter")
	repeat := parseStrList(items[4], "repeat")
	return &AtlasHeader{
		Image:   strings.TrimSpace(items[0]),
		W:       size[0],
		H:       size[1],
		Format:  format[0],
		WFilter: filter[0],
		HFilter: filter[1],
		Repeat:  repeat[0],
	}
}

func parseStrList(line string, name string) []string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, name) {
		panic(fmt.Sprintf("%s not start with %s", line, name))
	}
	items := strings.Split(line[len(name)+2:], ",")
	res := make([]string, 0)
	for _, item := range items {
		res = append(res, strings.TrimSpace(item))
	}
	return res
}

func parseIntList(line string, name string) []int {
	items := parseStrList(line, name)
	res := make([]int, 0)
	for _, item := range items {
		val, err := strconv.ParseInt(item, 10, 64)
		HandleErr(err)
		res = append(res, int(val))
	}
	return res
}

func parseBoolList(line string, name string) []bool {
	items := parseStrList(line, name)
	res := make([]bool, 0)
	for _, item := range items {
		val, err := strconv.ParseBool(item)
		HandleErr(err)
		res = append(res, val)
	}
	return res
}
