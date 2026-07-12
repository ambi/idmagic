import { defineDictionary } from '../../lib/i18n'

export const devicePageDictionary = defineDictionary(
  {
    asideTitle: '新しいデバイスを、安全な確認手順で接続。',
    asideText: '表示されたコードと接続先を確認し、自分が開始した操作だけを承認してください。',
    eyebrow: 'デバイス認可',
    title: 'デバイスを接続',
    description: '接続するデバイスに表示されている8文字のコードを入力してください。',
    codeLabel: 'デバイスコード',
    codeHintPrefix: 'ハイフンは入力不要です。例: ',
    warningNote: 'コードが一致していても、自分で開始していない接続要求は承認しないでください。',
    processing: '処理しています…',
    approve: 'このデバイスを承認',
    deny: '接続を拒否',
    deviceError: 'デバイス要求を処理できませんでした。',
  },
  {
    asideTitle: 'Connect a new device with a secure verification step.',
    asideText:
      'Check the displayed code and destination, and approve only the request you started yourself.',
    eyebrow: 'Device authorization',
    title: 'Connect a device',
    description: 'Enter the 8-character code shown on the device you are connecting.',
    codeLabel: 'Device code',
    codeHintPrefix: 'No need to type hyphens. Example: ',
    warningNote:
      'Even if the code matches, do not approve a connection request you did not start yourself.',
    processing: 'Processing…',
    approve: 'Approve this device',
    deny: 'Deny connection',
    deviceError: 'Could not process the device request.',
  },
)
