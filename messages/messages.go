package messages

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/iotku/mumzic/config"
	"github.com/nfnt/resize"
)

// Default Message size limits for Murmur
const (
	MAX_MESSAGE_LENGTH_WITH_IMAGE    = 131072
	MAX_MESSAGE_LENGTH_WITHOUT_IMAGE = 5000
)

type MessageTable struct {
	table *strings.Builder
	cols  int
}

var messageBuffers = make(map[string][]string)
var messageOffsets = make(map[string]int)

func ResetMore(sender string) {
	messageBuffers[sender] = nil
	messageOffsets[sender] = 0
}

func SendMore(sender, text string) {
	messageBuffers[sender] = append(messageBuffers[sender], text)
}

// SaveMoreRows adds the first rows limited by config.MaxLines to the provided
// table and then saves the additional rows into the more buffer
func SaveMoreRows(sender string, rows []string, table MessageTable) int {
	ResetMore(sender)
	var i int
	for i = 0; i < config.MaxLines && i < len(rows); i++ {
		table.AddRow(strconv.Itoa(i) + ": " + rows[i])
		SendMore(sender, strconv.Itoa(i)+": "+rows[i])
	}

	var extra int
	for ; i < len(rows); i++ {
		SendMore(sender, strconv.Itoa(i)+": "+rows[i])
		extra++
	}
	if extra != 0 {
		messageOffsets[sender] = config.MaxLines
		table.AddRow("---")
		table.AddRow("There are " + strconv.Itoa(extra) + " additional results.")
		table.AddRow("Use <b>more</b> and <b>less</b> to see them.")
	}

	return extra
}

func GetMore(sender string) (output []string) {
	offset := messageOffsets[sender]
	if offset == len(messageBuffers[sender]) {
		return []string{"nothing more"}
	}
	var i int
	for i = offset; i < offset+config.MaxLines && i < len(messageBuffers[sender]); i++ {
		output = append(output, messageBuffers[sender][i])
	}
	if i <= len(messageBuffers[sender]) {
		messageOffsets[sender] = i
	}
	return
}

func GetMoreTable(sender string) string {
	table := MakeTable("More Results")
	for _, v := range GetMore(sender) {
		table.AddRow(v)
	}
	return table.String()
}

func GetLess(sender string) (output []string) { // TODO: Investigate offsets not always being correct
	offset := messageOffsets[sender] - config.MaxLines
	if offset+config.MaxLines <= 0 {
		return []string{"Nothing less"}
	}

	if offset-config.MaxLines < 0 {
		offset = 0
	}

	var i int
	for i = offset; i < offset+config.MaxLines && i < len(messageBuffers[sender]); i++ {
		output = append(output, messageBuffers[sender][i])
	}
	if offset <= config.MaxLines {
		messageOffsets[sender] = 0
	} else {
		messageOffsets[sender] -= config.MaxLines
	}
	return
}

func GetLessTable(sender string) string {
	table := MakeTable("Less Results")
	for _, v := range GetLess(sender) {
		table.AddRow(v)
	}
	return table.String()
}

// MakeTable generates a html table with the first parameter as a header on top of the table
// and subsequent as column headers for the table
func MakeTable(header string, columns ...string) MessageTable {
	var b strings.Builder
	fmt.Fprintf(&b, "<h2 style=\"margin-top:16px; margin-bottom:2px; margin-left:0px; margin-right:0px; -qt-block-indent:0; text-indent:0px;\"><b><u><span style=\"font-size:x-large\">%s</span></u></b></h2>", header)
	fmt.Fprintf(&b, "<table align=\"left\" border=\"0\" style=\"margin-top:0px; margin-bottom:0px; margin-left:0px; margin-right:0px;\" cellspacing=\"2\" cellpadding=\"0\"><thead>")
	if len(columns) != 0 {
		fmt.Fprintf(&b, "<tr>")
	}
	for _, v := range columns {
		fmt.Fprintf(&b, "<th align=\"left\">%s</th>", v)
	}
	if len(columns) != 0 {
		fmt.Fprintf(&b, "</tr>")
	}
	fmt.Fprintf(&b, "</thead><tbody>")
	return MessageTable{&b, len(columns)}
}

// AddRow adds cells to a MessageTable
func (msgTbl MessageTable) AddRow(cells ...string) {
	fmt.Fprintf(msgTbl.table, "<tr>")
	for _, v := range cells {
		fmt.Fprintf(msgTbl.table, "<td><p>%s</p></td>", v)
	}
	fmt.Fprintf(msgTbl.table, "</tr>")
}

// String escapes the tbody and table elements of a MessageTable and then returns a string of the MessageTable
func (msgTbl MessageTable) String() string {
	fmt.Fprintf(msgTbl.table, "</tbody></table>")
	return msgTbl.table.String()
}

func FindCoverArtPath(playPath string) string {
	basedir := filepath.Dir(playPath)
	// TODO: robust matching
	if _, err := os.Stat(basedir + "/cover.jpg"); err == nil {
		return basedir + "/cover.jpg"
	}
	if _, err := os.Stat(basedir + "/cover.png"); err == nil {
		return basedir + "/cover.png"
	}
	return ""
}

// GenerateCoverArtImg creates a base64 encoded html <img>
// TODO: Find a way to get generated cover art to follow the larger limits (for messages that contain images)
//       for now, we make sure the image is less than maxSize to be well below the 5000 text limit the mumble server
//       imposes by default for text messages (that contain no image)
func GenerateCoverArtImg(path string) string {
	img, err := decodeImage(path)
	if err != nil {
		log.Println("Failed to decode img: ", path, " ", err)
		return ""
	}
	resizedImg := resize.Resize(100, 100, img, resize.Lanczos3)
	jpegQuality := 60
	maxSize := 4000
	var buf bytes.Buffer
	var encodedStr string
	for maxSize >= 4000 && jpegQuality > 0 {
		buf.Reset()
		options := jpeg.Options{Quality: jpegQuality}
		if err := jpeg.Encode(&buf, resizedImg, &options); err != nil {
			log.Println("Error encoding jpg for base64: ", path, " ", err)
			return ""
		}
		encodedStr = "<img src=\"data:img/jpeg;base64, " + base64.StdEncoding.EncodeToString(buf.Bytes()) + "\" />"
		maxSize = len(encodedStr)
		jpegQuality -= 10 // Lower potential future jpegQuality if there's further passes
	}
	if len(encodedStr) > MAX_MESSAGE_LENGTH_WITHOUT_IMAGE-150 {
		return "" // Don't return album art, it's certainly too big!
	}
	return encodedStr
}

func decodeImage(path string) (image.Image, error) {
	// File path is currently controlled by song database which is considered a trusted source of information
	// This /should/ not change, but extra contingencies may be necessary if we start getting images from external sources
	// which could potentially have troublesome arbitrary filenames
	//#nosec G304
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	if err := f.Close(); err != nil {
		log.Fatal(err)
	}

	return img, nil
}
