package domain

import "strings"

// DeviceLabel は User-Agent からブラウザーと OS の系統だけを取り出した表示ラベルを返す
// (例 "Chrome / macOS")。信頼済みデバイスの一覧とセキュリティ通知の本文が共に使うため、
// 機能別ではなく Context 共通の domain に置く。自分の端末を見分けるのに要る粒度はここまで
// で、生の User-Agent を保存しても失効の判断には寄与せず、漏れたときの被害だけが増える。
// 判別できない場合は空文字を返し、呼び出し側は既定の表示へフォールバックする。
func DeviceLabel(userAgent string) string {
	browser, os := browserFamily(userAgent), osFamily(userAgent)
	switch {
	case browser != "" && os != "":
		return browser + " / " + os
	case browser != "":
		return browser
	default:
		return os
	}
}

// browserFamily は代表的なブラウザーの系統名を返す。判定順は包含関係の狭い方から並べる
// (Edge / Opera は Chrome を、Chrome は Safari を User-Agent に含むため)。
func browserFamily(userAgent string) string {
	for _, candidate := range []struct{ token, name string }{
		{"Edg/", "Edge"},
		{"OPR/", "Opera"},
		{"Firefox/", "Firefox"},
		{"Chrome/", "Chrome"},
		{"Safari/", "Safari"},
	} {
		if strings.Contains(userAgent, candidate.token) {
			return candidate.name
		}
	}
	return ""
}

// osFamily は OS の系統名を返す。iPadOS / iOS は同じ "iOS" にまとめる。機種や
// バージョンまでは持たない。
func osFamily(userAgent string) string {
	for _, candidate := range []struct{ token, name string }{
		{"iPhone", "iOS"},
		{"iPad", "iOS"},
		{"Android", "Android"},
		{"Windows", "Windows"},
		{"Mac OS X", "macOS"},
		{"CrOS", "ChromeOS"},
		{"Linux", "Linux"},
	} {
		if strings.Contains(userAgent, candidate.token) {
			return candidate.name
		}
	}
	return ""
}
