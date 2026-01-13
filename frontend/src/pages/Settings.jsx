import { useState, useEffect } from 'react'
import { useAuth } from '../contexts/AuthContext'
import {
  Shield,
  Lock,
  Bell,
  Database,
  Zap,
  Save,
  AlertCircle,
  CheckCircle2,
  Settings as SettingsIcon
} from 'lucide-react'

const Settings = () => {
  const { user } = useAuth()
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState(null)

  // Настройки шифрования
  const [encryptionMode, setEncryptionMode] = useState('PLAIN')

  // Настройки уведомлений
  const [notifications, setNotifications] = useState({
    email: true,
    telegram: false,
    desktop: true,
  })

  // Настройки ML модели
  const [mlSettings, setMlSettings] = useState({
    confidenceThreshold: 0.70,
  })

  // Настройки контроля доступа
  const [accessControl, setAccessControl] = useState({
    requireAccessRequest: true,
    autoApproveAdmins: false,
  })

  // Настройки системы
  const [systemSettings, setSystemSettings] = useState({
    autoRefreshInterval: 30,
    incidentsPerPage: 20,
    logLevel: 'INFO',
    enableAuditLog: true,
  })

  useEffect(() => {
    fetchSettings()
  }, [])

  const fetchSettings = async () => {
    try {
      const token = localStorage.getItem('token')
      const response = await fetch('http://localhost:8080/api/settings', {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })

      if (response.ok) {
        const data = await response.json()
        // Update state with fetched settings
        if (data.accessControl) {
          setAccessControl({
            requireAccessRequest: data.accessControl.requireAccessRequest,
            autoApproveAdmins: data.accessControl.autoApproveAdmins || false
          })
        }
      }
    } catch (error) {
      console.error('Failed to fetch settings:', error)
    }
  }

  const handleSave = async () => {
    setSaving(true)
    setMessage(null)

    try {
      const token = localStorage.getItem('token')
      const response = await fetch('http://localhost:8080/api/settings', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          accessControl: {
            requireAccessRequest: accessControl.requireAccessRequest,
            autoApproveAdmins: accessControl.autoApproveAdmins
          }
        })
      })

      if (response.ok) {
        setMessage({ type: 'success', text: 'Настройки успешно сохранены' })
      } else {
        const error = await response.json()
        throw new Error(error.error || 'Ошибка при сохранении настроек')
      }
    } catch (error) {
      setMessage({ type: 'error', text: error.message || 'Ошибка при сохранении настроек' })
    } finally {
      setSaving(false)
    }
  }

  // Проверка прав доступа (только админ)
  const isAdmin = user?.role === 'admin'

  // DEBUG: Временно выводим информацию о пользователе
  console.log('User object:', user)
  console.log('User role:', user?.role)
  console.log('Is admin:', isAdmin)

  // Временно отключаем проверку для отладки
  // if (!isAdmin) {
  //   return (
  //     <div className="max-w-4xl mx-auto">
  //       <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-6">
  //         <div className="flex items-center gap-3">
  //           <AlertCircle className="w-6 h-6 text-yellow-600 dark:text-yellow-400" />
  //           <div>
  //             <h3 className="font-semibold text-yellow-900 dark:text-yellow-200">
  //               Доступ ограничен
  //             </h3>
  //             <p className="text-sm text-yellow-700 dark:text-yellow-300 mt-1">
  //               Только администраторы могут изменять системные настройки
  //             </p>
  //           </div>
  //         </div>
  //       </div>
  //     </div>
  //   )
  // }

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white flex items-center gap-3">
            <SettingsIcon className="w-8 h-8" />
            Настройки системы
          </h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            Управление параметрами безопасности и работы системы
          </p>
        </div>
      </div>

      {/* Message */}
      {message && (
        <div
          className={`p-4 rounded-lg border flex items-center gap-3 ${
            message.type === 'success'
              ? 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800 text-green-800 dark:text-green-200'
              : 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800 text-red-800 dark:text-red-200'
          }`}
        >
          {message.type === 'success' ? (
            <CheckCircle2 className="w-5 h-5" />
          ) : (
            <AlertCircle className="w-5 h-5" />
          )}
          <span>{message.text}</span>
        </div>
      )}

      {/* Encryption Settings */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
        <div className="flex items-center gap-3 mb-6">
          <div className="w-10 h-10 bg-primary-100 dark:bg-primary-900/30 rounded-lg flex items-center justify-center">
            <Lock className="w-6 h-6 text-primary-600 dark:text-primary-400" />
          </div>
          <div>
            <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
              Шифрование данных
            </h2>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              Выбор алгоритма для защиты персональных данных
            </p>
          </div>
        </div>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
              Алгоритм шифрования
            </label>
            <div className="space-y-3">
              {/* AES-256 */}
              <label className="flex items-start gap-3 p-4 border-2 rounded-lg cursor-pointer transition-all hover:bg-gray-50 dark:hover:bg-gray-700/50 relative">
                <input
                  type="radio"
                  name="encryption"
                  value="AES256"
                  checked={encryptionMode === 'AES256'}
                  onChange={(e) => setEncryptionMode(e.target.value)}
                  className="mt-1"
                />
                <div className="flex-1">
                  <div className="font-medium text-gray-900 dark:text-white">
                    AES-256-GCM
                    <span className="ml-2 text-xs bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 px-2 py-0.5 rounded">
                      Международный стандарт
                    </span>
                  </div>
                  <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                    • Высокая производительность (~10 GB/sec)<br />
                    • Аппаратное ускорение (AES-NI)<br />
                    • Стандарт FIPS 197, NIST SP 800-38D
                  </p>
                </div>
              </label>

              {/* GOST */}
              <label className="flex items-start gap-3 p-4 border-2 rounded-lg cursor-pointer transition-all hover:bg-gray-50 dark:hover:bg-gray-700/50">
                <input
                  type="radio"
                  name="encryption"
                  value="GOST"
                  checked={encryptionMode === 'GOST'}
                  onChange={(e) => setEncryptionMode(e.target.value)}
                  className="mt-1"
                />
                <div className="flex-1">
                  <div className="font-medium text-gray-900 dark:text-white">
                    ГОСТ Р 34.12-2015 "Кузнечик"
                    <span className="ml-2 text-xs bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300 px-2 py-0.5 rounded">
                      Российский стандарт
                    </span>
                  </div>
                  <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                    • Соответствие ГОСТ Р 34.12-2015<br />
                    • Сертификация ФСТЭК/ФСБ России<br />
                    • Обязателен для государственных систем РФ
                  </p>
                </div>
              </label>

              {/* PLAIN (dev only) */}
              <label className="flex items-start gap-3 p-4 border-2 border-yellow-300 dark:border-yellow-700 rounded-lg cursor-pointer transition-all hover:bg-yellow-50 dark:hover:bg-yellow-900/20">
                <input
                  type="radio"
                  name="encryption"
                  value="PLAIN"
                  checked={encryptionMode === 'PLAIN'}
                  onChange={(e) => setEncryptionMode(e.target.value)}
                  className="mt-1"
                />
                <div className="flex-1">
                  <div className="font-medium text-gray-900 dark:text-white">
                    Без шифрования (PLAIN)
                    <span className="ml-2 text-xs bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300 px-2 py-0.5 rounded">
                      Только для разработки
                    </span>
                  </div>
                  <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                    ⚠️ Не использовать в production! Данные хранятся в открытом виде
                  </p>
                </div>
              </label>
            </div>
          </div>

          {encryptionMode !== 'PLAIN' && (
            <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
              <p className="text-sm text-blue-800 dark:text-blue-200">
                <strong>⚠️ Важно:</strong> Изменение алгоритма шифрования требует миграции всех существующих данных.
                Это может занять несколько минут и потребовать остановки системы.
              </p>
            </div>
          )}
        </div>
      </div>

      {/* Notifications Settings */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
        <div className="flex items-center gap-3 mb-6">
          <div className="w-10 h-10 bg-orange-100 dark:bg-orange-900/30 rounded-lg flex items-center justify-center">
            <Bell className="w-6 h-6 text-orange-600 dark:text-orange-400" />
          </div>
          <div>
            <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
              Уведомления
            </h2>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              Настройка каналов оповещения
            </p>
          </div>
        </div>

        <div className="space-y-4">
          <label className="flex items-center justify-between p-4 border rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700/50 cursor-pointer">
            <div>
              <div className="font-medium text-gray-900 dark:text-white">Email уведомления</div>
              <p className="text-sm text-gray-600 dark:text-gray-400">
                Отправка писем о новых инцидентах
              </p>
            </div>
            <input
              type="checkbox"
              checked={notifications.email}
              onChange={(e) => setNotifications({ ...notifications, email: e.target.checked })}
              className="w-5 h-5"
            />
          </label>

          <label className="flex items-center justify-between p-4 border rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700/50 cursor-pointer">
            <div>
              <div className="font-medium text-gray-900 dark:text-white">Telegram Bot</div>
              <p className="text-sm text-gray-600 dark:text-gray-400">
                Уведомления через Telegram бот
              </p>
            </div>
            <input
              type="checkbox"
              checked={notifications.telegram}
              onChange={(e) => setNotifications({ ...notifications, telegram: e.target.checked })}
              className="w-5 h-5"
            />
          </label>

          <label className="flex items-center justify-between p-4 border rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700/50 cursor-pointer">
            <div>
              <div className="font-medium text-gray-900 dark:text-white">Desktop уведомления</div>
              <p className="text-sm text-gray-600 dark:text-gray-400">
                Всплывающие уведомления в браузере
              </p>
            </div>
            <input
              type="checkbox"
              checked={notifications.desktop}
              onChange={(e) => setNotifications({ ...notifications, desktop: e.target.checked })}
              className="w-5 h-5"
            />
          </label>
        </div>
      </div>

      {/* Access Control Settings */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
        <div className="flex items-center gap-3 mb-6">
          <div className="w-10 h-10 bg-purple-100 dark:bg-purple-900/30 rounded-lg flex items-center justify-center">
            <Shield className="w-6 h-6 text-purple-600 dark:text-purple-400" />
          </div>
          <div>
            <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
              Контроль доступа к данным
            </h2>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              Управление доступом к содержимому сообщений (152-ФЗ)
            </p>
          </div>
        </div>

        <div className="space-y-4">
          <label className="flex items-start gap-3 p-4 border-2 rounded-lg cursor-pointer transition-all hover:bg-gray-50 dark:hover:bg-gray-700/50">
            <input
              type="checkbox"
              checked={accessControl.requireAccessRequest}
              onChange={(e) => setAccessControl({ ...accessControl, requireAccessRequest: e.target.checked })}
              className="w-5 h-5 mt-0.5"
            />
            <div className="flex-1">
              <div className="font-medium text-gray-900 dark:text-white mb-1">
                Требовать запрос доступа для просмотра содержимого
              </div>
              <p className="text-sm text-gray-600 dark:text-gray-400">
                Операторы не смогут видеть текст сообщений без одобрения администратора.
                Содержимое будет отображаться как "∗∗∗∗∗∗∗" до получения доступа.
              </p>
              <div className="mt-3 p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                <p className="text-xs text-blue-800 dark:text-blue-200">
                  <strong>Рекомендуется:</strong> Включать для соответствия требованиям 152-ФЗ
                  "О персональных данных". Обеспечивает полный аудит доступа к конфиденциальной информации.
                </p>
              </div>
            </div>
          </label>

          {accessControl.requireAccessRequest && (
            <label className="flex items-start gap-3 p-4 border rounded-lg cursor-pointer transition-all hover:bg-gray-50 dark:hover:bg-gray-700/50 ml-8">
              <input
                type="checkbox"
                checked={accessControl.autoApproveAdmins}
                onChange={(e) => setAccessControl({ ...accessControl, autoApproveAdmins: e.target.checked })}
                className="w-5 h-5 mt-0.5"
              />
              <div className="flex-1">
                <div className="font-medium text-gray-900 dark:text-white mb-1">
                  Автоматически одобрять запросы от администраторов
                </div>
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  Администраторы получат доступ к содержимому сразу, без ожидания одобрения.
                  Все действия будут записаны в журнал аудита.
                </p>
              </div>
            </label>
          )}

          {!accessControl.requireAccessRequest && (
            <div className="p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg">
              <div className="flex gap-3">
                <AlertCircle className="w-5 h-5 text-yellow-600 dark:text-yellow-400 flex-shrink-0 mt-0.5" />
                <div className="text-sm text-yellow-800 dark:text-yellow-200">
                  <strong>⚠️ Предупреждение:</strong> При отключении контроля доступа все операторы
                  смогут видеть полное содержимое сообщений без ограничений. Это может не соответствовать
                  требованиям защиты персональных данных.
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* ML Settings */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
        <div className="flex items-center gap-3 mb-6">
          <div className="w-10 h-10 bg-indigo-100 dark:bg-indigo-900/30 rounded-lg flex items-center justify-center">
            <Zap className="w-6 h-6 text-indigo-600 dark:text-indigo-400" />
          </div>
          <div>
            <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
              Настройки ML модели
            </h2>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              Параметры классификации угроз
            </p>
          </div>
        </div>

        <div className="space-y-4">
          <div className="p-4 border rounded-lg">
            <label className="block">
              <div className="flex justify-between mb-2">
                <span className="font-medium text-gray-900 dark:text-white">
                  Порог уверенности
                </span>
                <span className="text-sm text-gray-600 dark:text-gray-400">
                  {(mlSettings.confidenceThreshold * 100).toFixed(0)}%
                </span>
              </div>
              <input
                type="range"
                min="0.5"
                max="0.95"
                step="0.05"
                value={mlSettings.confidenceThreshold}
                onChange={(e) => setMlSettings({ ...mlSettings, confidenceThreshold: parseFloat(e.target.value) })}
                className="w-full"
              />
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
                Минимальная уверенность модели для создания инцидента. Рекомендуется: 70-80%
              </p>
            </label>
          </div>

          <div className="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg border border-gray-200 dark:border-gray-600">
            <div className="flex items-start gap-3">
              <Zap className="w-5 h-5 text-indigo-600 dark:text-indigo-400 flex-shrink-0 mt-0.5" />
              <div className="text-sm text-gray-700 dark:text-gray-300">
                <strong>Текущая модель:</strong> ruBERT (DeepPavlov)<br />
                <strong>Точность:</strong> 67.96%<br />
                <strong>Датасет:</strong> ~16,000 размеченных сообщений<br />
                <strong>Категорий:</strong> 9 (8 угроз + нейтральное)
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* System Settings */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
        <div className="flex items-center gap-3 mb-6">
          <div className="w-10 h-10 bg-gray-100 dark:bg-gray-700 rounded-lg flex items-center justify-center">
            <Database className="w-6 h-6 text-gray-600 dark:text-gray-400" />
          </div>
          <div>
            <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
              Системные настройки
            </h2>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              Общие параметры работы системы
            </p>
          </div>
        </div>

        <div className="space-y-4">
          <div className="p-4 border rounded-lg">
            <label className="block">
              <div className="font-medium text-gray-900 dark:text-white mb-2">
                Интервал автообновления (секунды)
              </div>
              <input
                type="number"
                min="10"
                max="300"
                value={systemSettings.autoRefreshInterval}
                onChange={(e) => setSystemSettings({ ...systemSettings, autoRefreshInterval: parseInt(e.target.value) })}
                className="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600"
              />
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
                Частота обновления данных на дашборде
              </p>
            </label>
          </div>

          <div className="p-4 border rounded-lg">
            <label className="block">
              <div className="font-medium text-gray-900 dark:text-white mb-2">
                Инцидентов на странице
              </div>
              <select
                value={systemSettings.incidentsPerPage}
                onChange={(e) => setSystemSettings({ ...systemSettings, incidentsPerPage: parseInt(e.target.value) })}
                className="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600"
              >
                <option value={10}>10</option>
                <option value={20}>20</option>
                <option value={50}>50</option>
                <option value={100}>100</option>
              </select>
            </label>
          </div>

          <div className="p-4 border rounded-lg">
            <label className="block">
              <div className="font-medium text-gray-900 dark:text-white mb-2">
                Уровень логирования
              </div>
              <select
                value={systemSettings.logLevel}
                onChange={(e) => setSystemSettings({ ...systemSettings, logLevel: e.target.value })}
                className="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600"
              >
                <option value="DEBUG">DEBUG (подробный)</option>
                <option value="INFO">INFO (обычный)</option>
                <option value="WARNING">WARNING (предупреждения)</option>
                <option value="ERROR">ERROR (только ошибки)</option>
              </select>
            </label>
          </div>

          <label className="flex items-center justify-between p-4 border rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700/50 cursor-pointer">
            <div>
              <div className="font-medium text-gray-900 dark:text-white">Журнал аудита</div>
              <p className="text-sm text-gray-600 dark:text-gray-400">
                Логирование всех действий пользователей (требование 152-ФЗ)
              </p>
            </div>
            <input
              type="checkbox"
              checked={systemSettings.enableAuditLog}
              onChange={(e) => setSystemSettings({ ...systemSettings, enableAuditLog: e.target.checked })}
              className="w-5 h-5"
            />
          </label>
        </div>
      </div>

      {/* Save Button */}
      <div className="flex justify-end gap-3">
        <button
          onClick={handleSave}
          disabled={saving}
          className="flex items-center gap-2 px-6 py-3 bg-primary-600 hover:bg-primary-700 disabled:bg-gray-400 text-white rounded-lg font-medium transition-colors"
        >
          {saving ? (
            <>
              <div className="animate-spin rounded-full h-5 w-5 border-2 border-white border-t-transparent" />
              Сохранение...
            </>
          ) : (
            <>
              <Save className="w-5 h-5" />
              Сохранить настройки
            </>
          )}
        </button>
      </div>
    </div>
  )
}

export default Settings
