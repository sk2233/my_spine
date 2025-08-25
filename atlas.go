package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type AtlasItem struct {
	Name         string
	Rotate       int
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
	rotate := parseRotate(items[1])
	xy := parseIntList(items[2], "xy")
	size := parseIntList(items[3], "size")
	orig := parseIntList(items[4], "orig")
	if rotate == 90 || rotate == 270 {
		size[0], size[1] = size[1], size[0]
		orig[0], orig[1] = orig[1], orig[0]
	}
	offset := parseIntList(items[5], "offset")
	index := parseIntList(items[6], "index")
	return &AtlasItem{
		Name:   strings.TrimSpace(items[0]),
		Rotate: rotate,
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

// 0  90  270
func parseRotate(item string) int {
	item = strings.TrimSpace(item)
	if !strings.HasPrefix(item, "rotate") {
		panic(fmt.Sprintf("%s is not a valid rotate", item))
	}
	item = strings.TrimSpace(item[8:])
	res, err := strconv.ParseBool(item) // 先尝试 bool 值
	if err == nil {
		if res {
			return 90
		}
		return 0
	}
	temp, err := strconv.ParseInt(item, 10, 64)
	HandleErr(err)
	return int(temp) // 再尝试 数字
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
