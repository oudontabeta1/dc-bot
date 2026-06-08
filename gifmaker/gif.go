package gifmaker

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func ConvertToGif(attachmentURL string) (file io.Reader, err error) {

	res, err := http.DefaultClient.Get(attachmentURL)

	if err != nil {
		log.Printf("Failed to get file: %v", err)
	}

	defer res.Body.Close()

	err = os.MkdirAll("./gif", 0755)
	if err != nil {
		log.Printf("Failed to create directory %v", err)
	}

	filePath := "./gif/in.png"
	out, err := os.Create(filePath)
	if err != nil {
		log.Printf("Failed to create file: %v", err)
		return nil, err
	}
	defer out.Close()

	// 2. HTTPレスポンスのBodyをファイルにコピー（書き込み）する
	written, err := io.Copy(out, res.Body)
	if err != nil {
		log.Printf("Failed to save image: %v", err)
		return nil, err
	}

	log.Printf("Successfully saved %d bytes to %s", written, filePath)

	// 一時ファイルのパスを設定
	palettePath := "./gif/palette.png"
	tmpOutput := "./gif/out.gif"

	// 処理完了後にパレット用の一時画像は確実に削除する
	defer os.Remove(palettePath)

	// 1. palettegen パス: 最適なパレット画像(256色)を生成
	fmt.Println("[1/2] 最適なカラーパレットを生成中...")
	err = ffmpeg.Input(filePath).
		Output(palettePath, ffmpeg.KwArgs{"vf": "palettegen"}).
		OverWriteOutput().
		Run()
	if err != nil {
		return nil, fmt.Errorf("palettegen エラー: %w", err)
	}

	// 2. paletteuse パス: 生成したパレットを使って最高画質でGIF化
	fmt.Println("[2/2] パレットを適用してGIFを出力中...")
	err = ffmpeg.Filter(
		[]*ffmpeg.Stream{
			ffmpeg.Input(filePath),
			ffmpeg.Input(palettePath),
		},
		"paletteuse",
		ffmpeg.Args{},
	).
		Output(tmpOutput, ffmpeg.KwArgs{"loop": "0"}).
		OverWriteOutput().
		Run()

	if err != nil {
		return nil, fmt.Errorf("paletteuse エラー: %w", err)
	}
	gif, err := os.Open("./gif/out.gif")
	if err != nil {
		return nil, fmt.Errorf("readfile エラー: %w", err)
	}

	return gif, nil
}
