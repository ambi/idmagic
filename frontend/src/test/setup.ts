import '@testing-library/jest-dom'

// jsdom の Blob/File は text() を実装しないため、FileReader 経由で補う。
if (typeof File !== 'undefined' && typeof File.prototype.text !== 'function') {
  File.prototype.text = function (this: File) {
    return new Promise<string>((resolve, reject) => {
      const reader = new FileReader()
      reader.onload = () => resolve(String(reader.result))
      reader.onerror = () => reject(reader.error)
      reader.readAsText(this)
    })
  }
}
