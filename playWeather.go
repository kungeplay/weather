package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func main() {
	weatherTexts, err := getWeatherDetailText()
	if err != nil {
		fmt.Println("getWeatherDetailText err", err)
		return
	}
	//欢迎词
	prologueTexts := genPrologueText()
	reportType := os.Args[1:]
	fmt.Println(os.Args[0], " args:", reportType)
	if len(reportType) >= 1 && reportType[0] == "tts" {
		reportWithCoquiAi(prologueTexts, weatherTexts)
	} else {
		reportWithPiper(prologueTexts, weatherTexts)
	}
}

func reportWithPiper(prologueTexts []string, weatherTexts []string) {
	texts := append(prologueTexts, weatherTexts...)
	setVolume(20)
	playAudio("/home/liujiakun/Data/thirdSoft/audio/call-to-attention-123107.wav")

	for i := range texts {
		playTextWithPiper(texts[i])
	}
	setVolume(30)
}

func reportWithCoquiAi(prologueTexts []string, weatherTexts []string) {
	wavFile := "/tmp/coqui_ai_weather" + time.Now().Format("20060102150405_000") + ".wav"
	weatherTexts = append(weatherTexts, "")
	str := strings.Join(weatherTexts, "。")
	genTtsWav(str, wavFile)

	setVolume(20)
	playAudio("/home/liujiakun/Data/thirdSoft/audio/call-to-attention-123107.wav")
	for i := range prologueTexts { //开场白中有时间信息，仍旧使用较快的piper
		playTextWithPiper(prologueTexts[i])
	}
	playAudio(wavFile)
	setVolume(30)
}

// coqui_ai 性能较差
func genTtsWav(content string, fileName string) {
	fmt.Println("gen content:" + content)
	// 注意需在外部调用环境(比如shell脚本)中设置TORCH_FORCE_NO_WEIGHTS_ONLY_LOAD="1"
	ttsCmd := exec.Command("/home/liujiakun/venv/tts_coqui_ai/bin/tts", "--text", content, "--model_name", "tts_models/zh-CN/baker/tacotron2-DDC-GST", "--vocoder_name", "vocoder_models/universal/libri-tts/fullband-melgan", "--out_path", fileName)
	var buffer bytes.Buffer
	ttsCmd.Stdout = &buffer
	if err := ttsCmd.Start(); err != nil {
		fmt.Println("tts command start error", err.Error())
		return
	}
	if err := ttsCmd.Wait(); err != nil {
		fmt.Println("tts command wait error", err.Error())
		return
	}
	fmt.Println("gen Tts Wav result:" + buffer.String())
}

// 播放音频，piper性能较好
func playTextWithPiper(content string) {
	fmt.Println("play content:" + content)

	echoCmd := exec.Command("echo", content)
	piperCmd := exec.Command("/home/liujiakun/Data/thirdSoft/piper/piper", "--model", "/home/liujiakun/Data/thirdSoft/piper/voices/zh_CN-huayan-medium.onnx", "--output-raw")
	aplayCmd := exec.Command("aplay", "-r", "22050", "-f", "S16_LE", "-t", "raw", "-")

	r1, w1 := io.Pipe()
	defer r1.Close()
	defer w1.Close()
	echoCmd.Stdout = w1
	piperCmd.Stdin = r1

	r2, w2 := io.Pipe()
	defer r2.Close()
	defer w2.Close()
	piperCmd.Stdout = w2
	aplayCmd.Stdin = r2

	var buffer bytes.Buffer
	aplayCmd.Stdout = &buffer

	if err := echoCmd.Start(); err != nil {
		fmt.Println("echo command start error", err.Error())
		return
	}
	if err := piperCmd.Start(); err != nil {
		fmt.Println("piper command start error", err.Error())
		return
	}
	if err := aplayCmd.Start(); err != nil {
		fmt.Println("aplay command start error", err.Error())
		return
	}
	if err := echoCmd.Wait(); err != nil {
		fmt.Println("echo command wait error", err.Error())
		return
	}
	if err := w1.Close(); err != nil {
		fmt.Println("pipe 1 close error", err.Error())
		return
	}
	if err := piperCmd.Wait(); err != nil {
		fmt.Println("piper command wait error", err.Error())
		return
	}
	if err := w2.Close(); err != nil {
		fmt.Println("pipe 2 close error", err.Error())
		return
	}
	if err := aplayCmd.Wait(); err != nil {
		fmt.Println("aplay command wait error", err.Error())
		return
	}

	fmt.Println("play result:" + buffer.String())
}

// 播放音频2
func playTextWithPiper2(content string) {
	fmt.Println("play content:" + content)

	//使用管道,生成音频文件
	echoCmd := exec.Command("echo", content)
	piperCmd := exec.Command("/home/liujiakun/Data/thirdSoft/piper/piper", "--model", "/home/liujiakun/Data/thirdSoft/piper/voices/zh_CN-huayan-medium.onnx", "--output_file", "/tmp/weatherAudio.wav")
	r1, w1 := io.Pipe()
	defer r1.Close()
	defer w1.Close()
	echoCmd.Stdout = w1
	piperCmd.Stdin = r1
	var piperBuffer bytes.Buffer
	piperCmd.Stdout = &piperBuffer
	if err := echoCmd.Start(); err != nil {
		fmt.Println("echo command start error", err)
		return
	}
	if err := piperCmd.Start(); err != nil {
		fmt.Println("piper command start error", err)
		return
	}
	if err := echoCmd.Wait(); err != nil {
		fmt.Println("echo command wait error", err)
		return
	}
	if err := w1.Close(); err != nil {
		fmt.Println("pipe 1 close error", err)
		return
	}
	if err := piperCmd.Wait(); err != nil {
		fmt.Println("piper command wait error", err)
		return
	}
	//播放音频
	playCmd := exec.Command("play", "/tmp/weatherAudio.wav")
	var playBuffer bytes.Buffer
	playCmd.Stdout = &playBuffer
	if err := playCmd.Start(); err != nil {
		fmt.Println("aplay command start error", err)
		return
	}
	if err := playCmd.Wait(); err != nil {
		fmt.Println("play command wait error", err.Error())
		return
	}
	fmt.Println("play result:" + playBuffer.String())
}

// 开场白
func genPrologueText() []string {
	now := time.Now()
	strCustom := now.Format("2006-01-02-15-04-05")
	split := strings.Split(strCustom, "-")
	currentHourStr := strings.TrimPrefix(split[3], "0")
	currentHour, err := strconv.Atoi(currentHourStr)
	if err != nil {
		fmt.Println("strconv.atoi error", err)
	}
	var prologueContents []string
	prologueContents = append(prologueContents, "嗨")
	if currentHour < 12 {
		prologueContents = append(prologueContents, "早上好")
	} else if currentHour < 18 {
		prologueContents = append(prologueContents, "中午好")
	} else {
		prologueContents = append(prologueContents, "晚上好")
	}
	prologueContents = append(prologueContents, "现在时间为"+arabicToChinese(split[0])+"年"+strings.TrimPrefix(split[1], "0")+"月"+strings.TrimPrefix(split[2], "0")+"日"+currentHourStr+"点"+strings.TrimPrefix(split[4], "0")+"分")
	return prologueContents
}

func setVolume(volume int) {
	//设置音量大小
	amixerCmd := exec.Command("amixer", "set", "Master", strconv.Itoa(volume)+"%")
	var amixerBuffer bytes.Buffer
	amixerCmd.Stdout = &amixerBuffer
	if err := amixerCmd.Start(); err != nil {
		fmt.Println("amixer command start error", err)
		return
	}
	if err := amixerCmd.Wait(); err != nil {
		fmt.Println("amixer command wait error", err.Error())
		return
	}
}

// 提示音，4
func playAudio(audioFile string) {
	//设置音量大小
	aplayCmd := exec.Command("aplay", audioFile)
	var playBuffer bytes.Buffer
	aplayCmd.Stdout = &playBuffer
	if err := aplayCmd.Start(); err != nil {
		fmt.Println("aplayCmd command start error", err)
		return
	}
	if err := aplayCmd.Wait(); err != nil {
		fmt.Println("aplayCmd command wait error", err)
		return
	}
}

func getWeatherDetailText() ([]string, error) {
	adcode := "110105"                //区域编码
	key := os.Getenv("A_MAP_LBS_KEY") //获取环境变量中查询天气开放平台的key
	liveWeather, err := queryWeather(adcode, BASE, key)
	if err != nil {
		fmt.Println("query live weather error", err)
		return nil, err
	}
	liveDetails := []string{}
	for i := range liveWeather.Lives {
		live := liveWeather.Lives[i]
		liveDetails = append(liveDetails, live.Province+live.City+"现在室外的天气是"+live.Weather)
		liveDetails = append(liveDetails, "气温为"+convertTemperature(live.Temperature)+"度")
		liveDetails = append(liveDetails, "风向朝"+live.Winddirection)
		liveDetails = append(liveDetails, "风力"+convertCompareSymbol(live.Windpower)+"级")
		liveDetails = append(liveDetails, "空气湿度为"+live.Humidity)
	}
	foreCaseWeather, err := queryWeather(adcode, ALL, key)
	if err != nil {
		fmt.Println("query foreCase weather error", err)
		return nil, err
	}
	casts := foreCaseWeather.Forecasts[0].Casts
	for i := 0; i < 3; i++ {
		foreCaseDate := "今天"
		if i == 1 {
			foreCaseDate = "明天"
		} else if i == 2 {
			foreCaseDate = "后天"
		}
		cast := casts[i]
		liveDetails = append(liveDetails, foreCaseDate+"是"+convertDateSymbol(cast.Date)+"，"+"星期"+strings.ReplaceAll(cast.Week, "7", "日"))
		liveDetails = append(liveDetails, foreCaseDate+"白天天气为"+cast.Dayweather+"天")
		liveDetails = append(liveDetails, "温度为"+convertTemperature(cast.Daytemp)+"度")
		liveDetails = append(liveDetails, foreCaseDate+"白天风向朝"+cast.Daywind)
		liveDetails = append(liveDetails, "风力"+convertCompareSymbol(cast.Daypower)+"级")

		liveDetails = append(liveDetails, foreCaseDate+"晚上天气为"+cast.Nightweather+"天")
		liveDetails = append(liveDetails, "温度为"+convertTemperature(cast.Nighttemp)+"度")
		liveDetails = append(liveDetails, foreCaseDate+"晚上的风向朝"+cast.Nightwind)
		liveDetails = append(liveDetails, "风力"+convertCompareSymbol(cast.Nightpower)+"级")
	}

	return liveDetails, nil
}

func convertDateSymbol(dateStr string) string { //2024-05-02
	dateSplit := strings.Split(dateStr, "-")
	return strings.TrimPrefix(dateSplit[1], "0") + "月" + strings.TrimPrefix(dateSplit[2], "0") + "号"

}

func convertTemperature(temperature string) string {
	return strings.ReplaceAll(temperature, "-", "零下")
}

func convertCompareSymbol(text string) string {
	text = strings.ReplaceAll(text, "≤", "小于等于")
	text = strings.ReplaceAll(text, "<", "小于")
	text = strings.ReplaceAll(text, "≥", "大于等于")
	text = strings.ReplaceAll(text, "＞", "大于")
	text = strings.ReplaceAll(text, "-", "到")
	return text
}

func arabicToChinese(text string) string {
	var chineseResult = ""
	for i := range text {
		temp := string(text[i])
		var chineseNum = ""
		switch temp {
		case "0":
			chineseNum = "零"
			break
		case "1":
			chineseNum = "一"
			break
		case "2":
			chineseNum = "二"
			break
		case "3":
			chineseNum = "三"
			break
		case "4":
			chineseNum = "四"
			break
		case "5":
			chineseNum = "五"
			break
		case "6":
			chineseNum = "六"
			break
		case "7":
			chineseNum = "七"
			break
		case "8":
			chineseNum = "八"
			break
		case "9":
			chineseNum = "九"
			break
		}
		chineseResult = chineseResult + chineseNum
	}
	return chineseResult
}

type Extension string

const (
	ALL  Extension = "all"
	BASE Extension = "base"
)

func queryWeather(abCode string, extension Extension, key string) (*WeatherInfoResp, error) {
	params := url.Values{}
	lastUrl, err := url.Parse("https://restapi.amap.com/v3/weather/weatherInfo")
	if err != nil {
		fmt.Println("url parse error", err)
		return nil, err
	}
	params.Set("key", key)
	params.Set("city", abCode)                  //城市编码
	params.Set("extensions", string(extension)) //气象类型,可选值：base/all base:返回实况天气 all:返回预报天气
	params.Set("output", "JSON")                //返回格式,可选值：JSON,XML
	lastUrl.RawQuery = params.Encode()
	urlPath := lastUrl.String()
	fmt.Println("query weather request url:" + urlPath)
	resp, err := http.Get(urlPath)
	if err != nil {
		fmt.Println("http get error", err)
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	infoResp := WeatherInfoResp{}
	err = json.Unmarshal(body, &infoResp)
	if err != nil {
		fmt.Println("json unmarshal error", err)
		return nil, err
	}
	fmt.Println("query weather resp:" + string(body))
	return &infoResp, nil
}

type WeatherInfoResp struct {
	Status    string     `json:"status"`    //返回状态,值为0或1 1：成功；0：失败
	Count     string     `json:"count"`     //返回结果总数目
	Info      string     `json:"info"`      //返回的状态信息
	Infocode  string     `json:"infocode"`  //返回状态说明,10000代表正确
	Lives     []Live     `json:"lives"`     //实况天气数据信息
	Forecasts []Forecast `json:"forecasts"` //预报天气信息数据
}
type Live struct {
	Province         string `json:"province"`      //省份名
	City             string `json:"city"`          //城市名
	Adcode           string `json:"adcode"`        //区域编码
	Weather          string `json:"weather"`       //天气现象（汉字描述）
	Temperature      string `json:"temperature"`   //实时气温，单位：摄氏度
	Winddirection    string `json:"winddirection"` //风向描述
	Windpower        string `json:"windpower"`     //风力级别，单位：级
	Humidity         string `json:"humidity"`      //空气湿度
	Reporttime       string `json:"reporttime"`    //数据发布的时间
	TemperatureFloat string `json:"temperature_float"`
	HumidityFloat    string `json:"humidity_float"`
}

type Forecast struct {
	City       string `json:"city"`       //城市名称
	Adcode     string `json:"adcode"`     //城市编码
	Province   string `json:"province"`   //省份名称
	Reporttime string `json:"reporttime"` //预报发布时间
	Casts      []Cast `json:"casts"`      //预报数据list结构，元素cast,按顺序为当天、第二天、第三天的预报数据
}

type Cast struct {
	Date           string `json:"date"`         //日期
	Week           string `json:"week"`         //星期几
	Dayweather     string `json:"dayweather"`   //白天天气现象
	Nightweather   string `json:"nightweather"` //晚上天气现象
	Daytemp        string `json:"daytemp"`      //白天温度
	Nighttemp      string `json:"nighttemp"`    //晚上温度
	Daywind        string `json:"daywind"`      //白天风向
	Nightwind      string `json:"nightwind"`    //晚上风向
	Daypower       string `json:"daypower"`     //白天风力
	Nightpower     string `json:"nightpower"`   //晚上风力
	DaytempFloat   string `json:"daytemp_float"`
	NighttempFloat string `json:"nighttemp_float"`
}
