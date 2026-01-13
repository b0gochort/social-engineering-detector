import { useEffect, useState } from 'react'
import { Settings, RefreshCw, CheckCircle, XCircle, ExternalLink, Copy, AlertTriangle, Save, Power } from 'lucide-react'

const CollectorSettings = () => {
  const [config, setConfig] = useState(null)
  const [loading, setLoading] = useState(true)
  const [testing, setTesting] = useState(false)
  const [restarting, setRestarting] = useState(false)
  const [saving, setSaving] = useState(false)

  // Telegram settings
  const [telegramConfig, setTelegramConfig] = useState({
    api_id: '',
    api_hash: '',
    phone: ''
  })

  // VK settings
  const [vkConfig, setVkConfig] = useState({
    app_id: '',
    access_token: ''
  })

  const [vkAuthURL, setVkAuthURL] = useState(null)
  const [loadingVKAuth, setLoadingVKAuth] = useState(false)

  // Telegram auth code
  const [telegramCode, setTelegramCode] = useState('')
  const [sendingCode, setSendingCode] = useState(false)

  useEffect(() => {
    fetchConfig()
  }, [])

  const fetchConfig = async () => {
    try {
      setLoading(true)
      const token = localStorage.getItem('token')
      const response = await fetch('http://localhost:8080/api/config/collector', {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })
      const data = await response.json()
      setConfig(data)
    } catch (error) {
      console.error('Failed to fetch config:', error)
    } finally {
      setLoading(false)
    }
  }

  const testConnection = async () => {
    try {
      setTesting(true)
      const token = localStorage.getItem('token')
      const response = await fetch('http://localhost:8080/api/config/collector/test', {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })
      const data = await response.json()

      if (data.connected) {
        alert('✅ Коллектор доступен!')
      } else {
        alert(`❌ Коллектор недоступен: ${data.error}`)
      }
    } catch (error) {
      alert(`❌ Ошибка подключения: ${error.message}`)
    } finally {
      setTesting(false)
    }
  }

  const saveConfig = async () => {
    try {
      setSaving(true)
      const token = localStorage.getItem('token')

      const configData = {
        telegram: {
          api_id: parseInt(telegramConfig.api_id) || 0,
          api_hash: telegramConfig.api_hash,
          phone: telegramConfig.phone
        },
        vk: {
          app_id: parseInt(vkConfig.app_id) || 0,
          access_token: vkConfig.access_token
        }
      }

      const response = await fetch('http://localhost:8080/api/config/collector/save', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(configData)
      })

      const data = await response.json()

      if (response.ok) {
        alert(`✅ ${data.message}`)
        fetchConfig()
      } else {
        throw new Error(data.error || 'Ошибка при сохранении конфигурации')
      }
    } catch (error) {
      alert(`❌ Ошибка: ${error.message}`)
    } finally {
      setSaving(false)
    }
  }

  const restartCollector = async () => {
    if (!confirm('Вы уверены что хотите перезапустить коллектор?')) {
      return
    }

    try {
      setRestarting(true)
      const token = localStorage.getItem('token')
      const response = await fetch('http://localhost:8080/api/config/collector/restart', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })

      const data = await response.json()

      if (response.ok) {
        alert(`✅ ${data.message}`)
        // Wait a bit and fetch config again
        setTimeout(fetchConfig, 3000)
      } else {
        throw new Error(data.error || 'Ошибка при перезапуске коллектора')
      }
    } catch (error) {
      alert(`❌ Ошибка: ${error.message}`)
    } finally {
      setRestarting(false)
    }
  }

  const getVKAuthURL = async () => {
    try {
      setLoadingVKAuth(true)
      const token = localStorage.getItem('token')
      const response = await fetch('http://localhost:8080/api/config/vk/auth-url', {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })

      if (!response.ok) {
        throw new Error('Не удалось получить OAuth URL')
      }

      const data = await response.json()
      setVkAuthURL(data)
    } catch (error) {
      alert(`❌ Ошибка: ${error.message}`)
    } finally {
      setLoadingVKAuth(false)
    }
  }

  const sendTelegramCode = async () => {
    if (!telegramCode.trim()) {
      alert('Введите код из Telegram')
      return
    }

    try {
      setSendingCode(true)
      const response = await fetch('http://localhost:8081/telegram/auth/code', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          code: telegramCode
        })
      })

      const data = await response.json()

      if (response.ok) {
        alert('✅ Код успешно отправлен! Telegram авторизован.')
        setTelegramCode('')
        // Refresh config to update status
        setTimeout(fetchConfig, 2000)
      } else {
        throw new Error(data.error || 'Ошибка при отправке кода')
      }
    } catch (error) {
      alert(`❌ Ошибка: ${error.message}`)
    } finally {
      setSendingCode(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white flex items-center gap-3">
            <Settings className="w-8 h-8" />
            Настройки коллектора
          </h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            Управление конфигурацией сборщика сообщений
          </p>
        </div>
        <div className="flex gap-3">
          <button
            onClick={testConnection}
            disabled={testing}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white rounded-lg transition-colors"
          >
            {testing ? (
              <div className="animate-spin rounded-full h-5 w-5 border-2 border-white border-t-transparent" />
            ) : (
              <RefreshCw className="w-5 h-5" />
            )}
            Проверить соединение
          </button>
        </div>
      </div>

      {/* Status Card */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">Статус</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="flex items-center gap-3 p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
            <div className={`w-3 h-3 rounded-full ${config?.telegram_configured ? 'bg-green-500' : 'bg-red-500'}`} />
            <div>
              <div className="text-sm text-gray-600 dark:text-gray-400">Telegram</div>
              <div className="font-medium text-gray-900 dark:text-white">
                {config?.telegram_configured ? 'Настроен' : 'Не настроен'}
              </div>
            </div>
          </div>

          <div className="flex items-center gap-3 p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
            <div className={`w-3 h-3 rounded-full ${config?.vk_configured ? 'bg-green-500' : 'bg-red-500'}`} />
            <div>
              <div className="text-sm text-gray-600 dark:text-gray-400">VK</div>
              <div className="font-medium text-gray-900 dark:text-white">
                {config?.vk_configured ? 'Настроен' : 'Не настроен'}
              </div>
            </div>
          </div>

          <div className="flex items-center gap-3 p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
            <div className="text-sm text-gray-600 dark:text-gray-400">URL коллектора</div>
            <div className="font-mono text-sm text-gray-900 dark:text-white">
              {config?.collector_url}
            </div>
          </div>
        </div>
      </div>

      {/* Telegram Configuration */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">Настройка Telegram</h2>
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              API ID
            </label>
            <input
              type="number"
              value={telegramConfig.api_id}
              onChange={(e) => setTelegramConfig({ ...telegramConfig, api_id: e.target.value })}
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              placeholder="Получите на my.telegram.org"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              API Hash
            </label>
            <input
              type="text"
              value={telegramConfig.api_hash}
              onChange={(e) => setTelegramConfig({ ...telegramConfig, api_hash: e.target.value })}
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              placeholder="API Hash от Telegram"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Номер телефона
            </label>
            <input
              type="tel"
              value={telegramConfig.phone}
              onChange={(e) => setTelegramConfig({ ...telegramConfig, phone: e.target.value })}
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              placeholder="+79XXXXXXXXX"
            />
          </div>

          <div className="bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg p-4">
            <p className="text-sm text-amber-800 dark:text-amber-200">
              <strong>⚠️ Важно:</strong> Получите API ID и Hash на странице
              <a href="https://my.telegram.org/apps" target="_blank" rel="noopener noreferrer" className="text-blue-600 dark:text-blue-400 underline ml-1">
                my.telegram.org/apps
              </a>
            </p>
          </div>

          {/* Telegram Auth Code Section */}
          <div className="border-t border-gray-200 dark:border-gray-700 pt-4 mt-4">
            <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-3">Авторизация Telegram</h3>
            <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">
              После сохранения конфигурации и перезапуска коллектора, Telegram отправит вам код подтверждения. Введите его ниже:
            </p>
            <div className="flex gap-3">
              <input
                type="text"
                value={telegramCode}
                onChange={(e) => setTelegramCode(e.target.value)}
                placeholder="Код из Telegram (например: 12345)"
                className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                onKeyPress={(e) => {
                  if (e.key === 'Enter') {
                    sendTelegramCode()
                  }
                }}
              />
              <button
                onClick={sendTelegramCode}
                disabled={sendingCode || !telegramCode.trim()}
                className="px-6 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white rounded-lg font-medium transition-colors flex items-center gap-2"
              >
                {sendingCode ? (
                  <div className="animate-spin rounded-full h-5 w-5 border-2 border-white border-t-transparent" />
                ) : (
                  <CheckCircle className="w-5 h-5" />
                )}
                Отправить код
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* VK Configuration */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">Настройка VK</h2>
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              App ID
            </label>
            <input
              type="number"
              value={vkConfig.app_id}
              onChange={(e) => setVkConfig({ ...vkConfig, app_id: e.target.value })}
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              placeholder="ID приложения VK"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Access Token
            </label>
            <textarea
              value={vkConfig.access_token}
              onChange={(e) => setVkConfig({ ...vkConfig, access_token: e.target.value })}
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono text-sm"
              placeholder="vk1.a.xxx..."
              rows={3}
            />
          </div>

          {/* VK OAuth Flow */}
          <div className="border-t border-gray-200 dark:border-gray-700 pt-4">
            <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">
              Или получите токен через OAuth:
            </p>
            <button
              onClick={getVKAuthURL}
              disabled={loadingVKAuth}
              className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white rounded-lg transition-colors"
            >
              {loadingVKAuth ? (
                <div className="animate-spin rounded-full h-5 w-5 border-2 border-white border-t-transparent" />
              ) : (
                <ExternalLink className="w-5 h-5" />
              )}
              Получить OAuth URL
            </button>

            {vkAuthURL && (
              <div className="mt-4 p-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
                <p className="text-sm text-blue-800 dark:text-blue-200 mb-2">
                  {vkAuthURL.instructions}
                </p>
                <div className="flex gap-2">
                  <input
                    type="text"
                    value={vkAuthURL.auth_url}
                    readOnly
                    className="flex-1 px-3 py-2 border border-blue-300 dark:border-blue-700 rounded-lg bg-white dark:bg-gray-800 text-sm font-mono"
                  />
                  <button
                    onClick={() => {
                      navigator.clipboard.writeText(vkAuthURL.auth_url)
                      alert('Скопировано!')
                    }}
                    className="px-3 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg"
                  >
                    <Copy className="w-5 h-5" />
                  </button>
                  <a
                    href={vkAuthURL.auth_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="px-3 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg"
                  >
                    <ExternalLink className="w-5 h-5" />
                  </a>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Action Buttons */}
      <div className="flex gap-3 justify-end">
        <button
          onClick={saveConfig}
          disabled={saving}
          className="flex items-center gap-2 px-6 py-3 bg-green-600 hover:bg-green-700 disabled:bg-gray-400 text-white rounded-lg font-medium transition-colors"
        >
          {saving ? (
            <div className="animate-spin rounded-full h-5 w-5 border-2 border-white border-t-transparent" />
          ) : (
            <Save className="w-5 h-5" />
          )}
          Сохранить конфигурацию
        </button>

        <button
          onClick={restartCollector}
          disabled={restarting}
          className="flex items-center gap-2 px-6 py-3 bg-orange-600 hover:bg-orange-700 disabled:bg-gray-400 text-white rounded-lg font-medium transition-colors"
        >
          {restarting ? (
            <div className="animate-spin rounded-full h-5 w-5 border-2 border-white border-t-transparent" />
          ) : (
            <Power className="w-5 h-5" />
          )}
          Перезапустить коллектор
        </button>
      </div>
    </div>
  )
}

export default CollectorSettings
