package messages

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"image"
	"image/jpeg"
	_ "image/png"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/iotku/mumzic/youtubedl"
	"github.com/nfnt/resize"
)

// Default Message size limits for Murmur
const (
	MaxMessageLengthWithImage    = 131072
	MaxMessageLengthWithoutImage = 5000
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
// table and then saves the additional rows into the 'more' buffer
func SaveMoreRows(sender string, maxLines int, rows []string, table MessageTable) int {
	ResetMore(sender)
	var i int
	for i = 0; i < maxLines && i < len(rows); i++ {
		table.AddRow(strconv.Itoa(i) + ": " + rows[i])
		SendMore(sender, strconv.Itoa(i)+": "+rows[i])
	}

	var extra int
	for ; i < len(rows); i++ {
		SendMore(sender, strconv.Itoa(i)+": "+rows[i])
		extra++
	}
	if extra != 0 {
		messageOffsets[sender] = maxLines
		table.AddRow("---")
		table.AddRow("There are " + strconv.Itoa(extra) + " additional results.")
		table.AddRow("Use <b>more</b> and <b>less</b> to see them.")
	}

	return extra
}

func GetMore(sender string, maxLines int) (output []string) {
	offset := messageOffsets[sender]
	if offset == len(messageBuffers[sender]) {
		return []string{"nothing more"}
	}
	var i int
	for i = offset; i < offset+maxLines && i < len(messageBuffers[sender]); i++ {
		output = append(output, messageBuffers[sender][i])
	}
	if i <= len(messageBuffers[sender]) {
		messageOffsets[sender] = i
	}
	return
}

func GetMoreTable(sender string, maxLines int) string {
	table := MakeTable("More Results")
	for _, v := range GetMore(sender, maxLines) {
		table.AddRow(v)
	}
	return table.String()
}

func GetLess(sender string, maxLines int) (output []string) { // TODO: Investigate offsets not always being correct
	offset := messageOffsets[sender] - maxLines
	if offset+maxLines <= 0 {
		return []string{"Nothing less"}
	}

	if offset-maxLines < 0 {
		offset = 0
	}

	var i int
	for i = offset; i < offset+maxLines && i < len(messageBuffers[sender]); i++ {
		output = append(output, messageBuffers[sender][i])
	}
	if offset <= maxLines {
		messageOffsets[sender] = 0
	} else {
		messageOffsets[sender] -= maxLines
	}
	return
}

func GetLessTable(sender string, maxLines int) string {
	table := MakeTable("Less Results")
	for _, v := range GetLess(sender, maxLines) {
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
//
//	for now, we make sure the image is less than maxSize to be well below the 5000 text limit the mumble server
//	imposes by default for text messages (that contain no image)
//
// TODO: Option to override limits for servers with modified settings
func GenerateCoverArtImg(image image.Image, maxBytes int) string {
	resizedImg := resize.Resize(0, 100, image, resize.MitchellNetravali)
	const minQual = 30
	const maxQual = 80

	var buf bytes.Buffer
	var best string
	low, high := minQual, maxQual
	for low <= high { // We love binary search
		mid := (low + high) / 2 // midpoint
		buf.Reset()
		if err := jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: mid}); err != nil {
			log.Println("Error encoding jpg in GenerateCoverArtImg: ", err)
			return ""
		}
		encodedStr := "<img src=\"data:img/jpeg;base64, " + base64.StdEncoding.EncodeToString(buf.Bytes()) + "\" />"

		if len(encodedStr) <= maxBytes {
			best = encodedStr
			low = mid + 1
		} else {
			high = mid - 1
		}
	}

	if len(best) > maxBytes {
		log.Println("[WARN] Generated album art was too big!")
		return "" // Don't return album art, it's certainly too big!
	}

	return best
}

func DecodeImage(path string) (image.Image, error) {
	if path == "" {
		return nil, errors.New("No path specified.")
	}

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

func NowPlaying(path, human string, isRadioMode bool, count int) string {
	header := "<h2><u>Now Playing</u></h2><table><tr><td>"
	var b strings.Builder
	b.WriteString(`</td><td><table><tr><td><a href="`)
	if strings.HasPrefix(path, "http") {
		b.WriteString(html.EscapeString(path))
	} else {
		b.WriteString(html.EscapeString("https://youtube.com/search?q=" + url.QueryEscape(human)))
	}
	b.WriteString(`">`)
	b.WriteString(html.EscapeString(human))
	b.WriteString(`</a></td></tr>`)

	if isRadioMode {
		b.WriteString(`<tr><td><b>Radio</b> Mode: <b>Enabled</b></td></tr>`)
	} else {
		fmt.Fprintf(&b, `<tr><td><b>%d</b> songs queued</td></tr>`, count)
	}
	b.WriteString(`</table></td></tr></table>`)

	// Generate Art Image in remaining space
	var artImg string
	var img image.Image
	var err error

	if strings.HasPrefix(path, "http") { // ytdlp thumbnail
		img, err = youtubedl.GetYtDLThumbnail(path)
	} else { // Local files
		img, err = DecodeImage(FindCoverArtPath(path))
	}

	if img != nil {
		artImg = GenerateCoverArtImg(img, MaxMessageLengthWithoutImage-len(header)-b.Len())
	}

	if err != nil {
		log.Println("Could not generate thumbnail: " + err.Error())
	}

	return header + artImg + b.String()
}
