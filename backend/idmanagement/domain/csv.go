package domain

// 種別に依存しない CSV 往復基盤。転送ポリシー、可逆なセル変換、行単位の解析器と
// 直列化器を持ち、User と Group の CSV はこれを共有する。種別ごとに違うのは
// 「どの機械キーを受理するか」だけなので、解析器はその判定だけを受け取る。
// 列の語彙と計画器は共有しない (docs/contexts/identity-management/internals.md)。

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"maps"
	"strconv"
	"strings"
)

// CSVTransferPolicy は組み立て境界から注入し、export と import が共有する。
// 上限は 1 個の成果物に対するものであり、テナントの母集団を制限しない。
type CSVTransferPolicy struct {
	MaxRows       int
	MaxBytes      int
	MaxFieldBytes int
}

func DefaultCSVTransferPolicy() CSVTransferPolicy {
	return CSVTransferPolicy{MaxRows: 100_000, MaxBytes: 64 << 20, MaxFieldBytes: 64 << 10}
}

func (p CSVTransferPolicy) Validate() error {
	if p.MaxRows <= 0 || p.MaxBytes <= 0 || p.MaxFieldBytes <= 0 {
		return errors.New("CSV transfer policy values must be positive")
	}
	return nil
}

type CSVErrorCode string

const (
	CSVErrorInvalidCSV         CSVErrorCode = "invalid_csv"
	CSVErrorInvalidHeader      CSVErrorCode = "invalid_header"
	CSVErrorCSVTooLarge        CSVErrorCode = "csv_too_large"
	CSVErrorTooManyRows        CSVErrorCode = "too_many_rows"
	CSVErrorFieldTooLarge      CSVErrorCode = "field_too_large"
	CSVErrorInvalidColumnCount CSVErrorCode = "invalid_column_count"
)

// CSVError は位置と安定コードだけを運ぶ。生のセル値がエラーとしてドメイン境界を
// 越えることは意図的に無い。
type CSVError struct {
	Row    int          `json:"row,omitempty"`
	Column string       `json:"column,omitempty"`
	Code   CSVErrorCode `json:"code"`
}

func (e CSVError) Error() string {
	return fmt.Sprintf("CSV error: row=%d column=%q code=%q", e.Row, e.Column, e.Code)
}

// CSVCell は列の存在とセルの内容を別々に保持する。列が無いことは Aggregate を
// 変更しない意味であり、存在する空セルは項目を消す意味を持ちうる。
type CSVCell struct {
	Present bool
	Raw     string
}

type CSVRow struct {
	Number int
	cells  map[string]CSVCell
}

func (r CSVRow) Cell(key string) (CSVCell, bool) {
	cell, ok := r.cells[key]
	return cell, ok
}

// TrimmedCell は存在する列の値を前後の空白を取り除いて返す。列が無ければ空文字列。
func (r CSVRow) TrimmedCell(key string) string {
	cell, ok := r.cells[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(cell.Raw)
}

// NewCSVRow は主にテスト用に、機械キーと値の対応から 1 行を組み立てる。
func NewCSVRow(number int, cells map[string]CSVCell) CSVRow {
	copied := make(map[string]CSVCell, len(cells))
	maps.Copy(copied, cells)
	return CSVRow{Number: number, cells: copied}
}

// forbiddenCSVHeaders はどの種別でもファイルごと拒否するヘッダー。秘密情報が
// CSV という便宜的な経路へ流れ込む余地を作らない。
var forbiddenCSVHeaders = map[string]struct{}{
	"password":      {},
	"password_hash": {},
	"mfa_secret":    {},
	"token":         {},
	"recovery_code": {},
}

type countingReader struct {
	r io.Reader
	n int64
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	r.n += int64(n)
	return n, err
}

type CSVRecord struct {
	Row   *CSVRow
	Error *CSVError
}

// CSVReader は RFC 4180 のレコードを逐次解析する。保持するのはヘッダーと現在の
// レコードだけで、バッチ化と結果の収集は呼び出し側が持つ。
type CSVReader struct {
	csv      *csv.Reader
	counting *countingReader
	header   []string
	policy   CSVTransferPolicy
	rowsRead int
	nextLine int
}

// NewCSVReader は accepts が受理する機械キーだけをヘッダーとして許す解析器を返す。
// accepts が種別ごとの唯一の差分であり、それ以外の規則は共有する。
func NewCSVReader(input io.Reader, accepts func(key string) bool, policy CSVTransferPolicy) (*CSVReader, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	if accepts == nil {
		return nil, errors.New("CSV reader requires a column acceptance rule")
	}
	limited := &io.LimitedReader{R: input, N: int64(policy.MaxBytes) + 1}
	counting := &countingReader{r: limited}
	r := csv.NewReader(counting)
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if counting.n > int64(policy.MaxBytes) {
		return nil, &CSVError{Row: 1, Code: CSVErrorCSVTooLarge}
	}
	if err != nil || len(header) == 0 {
		return nil, &CSVError{Row: 1, Code: CSVErrorInvalidCSV}
	}
	header[0] = strings.TrimPrefix(header[0], "\ufeff")
	seen := make(map[string]struct{}, len(header))
	for _, key := range header {
		if len(key) > policy.MaxFieldBytes {
			return nil, &CSVError{Row: 1, Column: key, Code: CSVErrorFieldTooLarge}
		}
		if _, forbidden := forbiddenCSVHeaders[key]; forbidden {
			return nil, &CSVError{Row: 1, Column: key, Code: CSVErrorInvalidHeader}
		}
		if !accepts(key) {
			return nil, &CSVError{Row: 1, Column: key, Code: CSVErrorInvalidHeader}
		}
		if _, duplicate := seen[key]; duplicate {
			return nil, &CSVError{Row: 1, Column: key, Code: CSVErrorInvalidHeader}
		}
		seen[key] = struct{}{}
	}
	return &CSVReader{csv: r, counting: counting, header: append([]string(nil), header...), policy: policy, nextLine: 2}, nil
}

func (r *CSVReader) Header() []string {
	return append([]string(nil), r.header...)
}

func (r *CSVReader) Next() (CSVRecord, error) {
	line := r.nextLine
	record, err := r.csv.Read()
	if r.counting.n > int64(r.policy.MaxBytes) {
		return CSVRecord{}, &CSVError{Row: line, Code: CSVErrorCSVTooLarge}
	}
	if errors.Is(err, io.EOF) {
		return CSVRecord{}, io.EOF
	}
	if err != nil {
		return CSVRecord{}, &CSVError{Row: line, Code: CSVErrorInvalidCSV}
	}
	r.nextLine++
	r.rowsRead++
	if r.rowsRead > r.policy.MaxRows {
		return CSVRecord{}, &CSVError{Row: line, Code: CSVErrorTooManyRows}
	}
	for i, raw := range record {
		if len(raw) > r.policy.MaxFieldBytes {
			column := ""
			if i < len(r.header) {
				column = r.header[i]
			}
			return CSVRecord{}, &CSVError{Row: line, Column: column, Code: CSVErrorFieldTooLarge}
		}
	}
	if len(record) != len(r.header) {
		return CSVRecord{Error: &CSVError{Row: line, Code: CSVErrorInvalidColumnCount}}, nil
	}
	row := &CSVRow{Number: line, cells: make(map[string]CSVCell, len(r.header))}
	for i, key := range r.header {
		row.cells[key] = CSVCell{Present: true, Raw: DecodeCSVCell(record[i])}
	}
	return CSVRecord{Row: row}, nil
}

func isCSVFormulaTrigger(value string) bool {
	if value == "" {
		return false
	}
	switch value[0] {
	case '=', '+', '-', '@', '\t', '\r', '\n':
		return true
	default:
		return false
	}
}

// EncodeCSVCell は表計算で危険な先頭文字と既存の先頭アポストロフィーに可逆な形で
// 接頭辞を付ける。DecodeCSVCell はその逆で、規定どおり 1 文字だけを取り除く。
func EncodeCSVCell(value string) string {
	if value != "" && (value[0] == '\'' || isCSVFormulaTrigger(value)) {
		return "'" + value
	}
	return value
}

func DecodeCSVCell(value string) string {
	if len(value) < 2 || value[0] != '\'' {
		return value
	}
	if value[1] == '\'' || isCSVFormulaTrigger(value[1:]) {
		return value[1:]
	}
	return value
}

type policyWriter struct {
	w       io.Writer
	written int
	max     int
}

func (w *policyWriter) Write(p []byte) (int, error) {
	if len(p) > w.max-w.written {
		return 0, &CSVError{Code: CSVErrorCSVTooLarge}
	}
	n, err := w.w.Write(p)
	w.written += n
	return n, err
}

// CSVWriter は 1 レコードずつ直列化し、CSVReader と同じポリシーを、先行する行を
// 保持せずに強制する。
type CSVWriter struct {
	csv       *csv.Writer
	headerLen int
	policy    CSVTransferPolicy
	rows      int
}

func NewCSVWriter(output io.Writer, header []string, policy CSVTransferPolicy) (*CSVWriter, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	if len(header) == 0 {
		return nil, &CSVError{Row: 1, Code: CSVErrorInvalidHeader}
	}
	seen := map[string]struct{}{}
	for _, key := range header {
		if len(key) > policy.MaxFieldBytes {
			return nil, &CSVError{Row: 1, Column: key, Code: CSVErrorFieldTooLarge}
		}
		if _, duplicate := seen[key]; duplicate {
			return nil, &CSVError{Row: 1, Column: key, Code: CSVErrorInvalidHeader}
		}
		seen[key] = struct{}{}
	}
	bounded := &policyWriter{w: output, max: policy.MaxBytes}
	w := &CSVWriter{csv: csv.NewWriter(bounded), headerLen: len(header), policy: policy}
	if err := w.writeRecord(header); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *CSVWriter) writeRecord(record []string) error {
	encoded := make([]string, len(record))
	for i, value := range record {
		encoded[i] = EncodeCSVCell(value)
	}
	return w.csv.Write(encoded)
}

func (w *CSVWriter) WriteRow(record []string) error {
	if len(record) != w.headerLen {
		return &CSVError{Row: w.rows + 2, Code: CSVErrorInvalidColumnCount}
	}
	if w.rows >= w.policy.MaxRows {
		return &CSVError{Row: w.rows + 2, Code: CSVErrorTooManyRows}
	}
	for i, value := range record {
		if len(value) > w.policy.MaxFieldBytes {
			return &CSVError{Row: w.rows + 2, Column: strconv.Itoa(i), Code: CSVErrorFieldTooLarge}
		}
	}
	if err := w.writeRecord(record); err != nil {
		return err
	}
	w.rows++
	return nil
}

func (w *CSVWriter) Close() error {
	w.csv.Flush()
	return w.csv.Error()
}
