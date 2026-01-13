import { useState } from 'react'
import { Zap, RefreshCw, Save, X, CheckCircle, AlertTriangle } from 'lucide-react'

const ModelTester = () => {
  const [text, setText] = useState('')
  const [result, setResult] = useState(null)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saveSuccess, setSaveSuccess] = useState(false)
  const [autoSave, setAutoSave] = useState(false)

  const categories = {
    1: 'Склонение к сексуальным действиям (Груминг)',
    2: 'Угрозы, шантаж, вымогательство',
    3: 'Физическое насилие/Буллинг',
    4: 'Склонение к суициду/Самоповреждению',
    5: 'Склонение к опасным играм/действиям',
    6: 'Пропаганда запрещенных веществ',
    7: 'Финансовое мошенничество',
    8: 'Сбор личных данных (Фишинг)',
    9: 'Нейтральное общение',
  }

  const testModels = async () => {
    if (!text.trim()) return

    setLoading(true)
    setSaveSuccess(false)
    try {
      const response = await fetch('http://localhost:8001/api/v1/classify/single', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ text: text.trim() })
      })

      if (!response.ok) throw new Error('Classification failed')

      const data = await response.json()
      setResult(data)

      // Автоматически сохраняем, если включен переключатель
      if (autoSave) {
        setTimeout(() => saveToDataset(data), 500)
      }
    } catch (error) {
      console.error('Failed to classify:', error)
      alert('Не удалось классифицировать текст')
    } finally {
      setLoading(false)
    }
  }

  const saveToDataset = async (dataToSave = null) => {
    const resultData = dataToSave || result
    if (!resultData) return

    setSaving(true)
    try {
      const token = localStorage.getItem('token')

      // Сохраняем в ML dataset с обеими категориями для анализа
      const justification = `Dual model test - v2: cat=${resultData.v2_prediction.category_id} conf=${Math.round(resultData.v2_prediction.confidence * 100)}%, v4: cat=${resultData.v4_prediction.category_id} conf=${Math.round(resultData.v4_prediction.confidence * 100)}%, agree=${resultData.models_agree}`

      const response = await fetch('http://localhost:8080/api/ml-dataset', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({
          message_text: text.trim(),
          category_id: resultData.primary_category_id,
          category_name: resultData.primary_category,
          justification: justification,
          provider: 'manual_test',
          model_version: 'dual_v2_v4',
          source: 'manual_testing'
        })
      })

      if (!response.ok) throw new Error('Failed to save')

      setSaveSuccess(true)
      setTimeout(() => setSaveSuccess(false), 3000)
    } catch (error) {
      console.error('Failed to save to dataset:', error)
      alert('Не удалось сохранить в датасет')
    } finally {
      setSaving(false)
    }
  }

  const reset = () => {
    setText('')
    setResult(null)
    setSaveSuccess(false)
  }

  const getCategoryColor = (categoryId) => {
    if (categoryId === 9) return 'success'
    if ([1, 2, 4, 5].includes(categoryId)) return 'danger'
    return 'warning'
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-gray-900">Тестирование моделей</h1>
        <p className="text-gray-500 mt-1">Проверьте работу обеих моделей (v2 и v4) на своих примерах</p>
      </div>

      {/* Input Section */}
      <div className="card">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-semibold">Введите текст для классификации</h2>

          {/* Auto-save toggle */}
          <label className="flex items-center gap-3 cursor-pointer">
            <span className="text-sm font-medium text-gray-700">
              Автоматически сохранять в датасет
            </span>
            <div className="relative">
              <input
                type="checkbox"
                checked={autoSave}
                onChange={(e) => setAutoSave(e.target.checked)}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary-300 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
            </div>
          </label>
        </div>

        <textarea
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="Например: Отправь мне свой пароль..."
          className="input min-h-[120px] mb-4"
          disabled={loading}
        />

        {autoSave && (
          <div className="mb-4 p-3 bg-blue-50 border border-blue-200 rounded-lg">
            <p className="text-sm text-blue-800">
              <span className="font-semibold">Режим автосохранения включен:</span> результаты тестирования будут автоматически сохраняться в датасет для обучения модели v5
            </p>
          </div>
        )}

        <div className="flex gap-3">
          <button
            onClick={testModels}
            disabled={loading || !text.trim()}
            className="btn-primary flex items-center gap-2 disabled:opacity-50"
          >
            {loading ? (
              <>
                <div className="animate-spin rounded-full h-5 w-5 border-2 border-white border-t-transparent"></div>
                Анализируем...
              </>
            ) : (
              <>
                <Zap className="w-5 h-5" />
                Протестировать
              </>
            )}
          </button>

          {result && (
            <button
              onClick={reset}
              className="btn-secondary flex items-center gap-2"
            >
              <RefreshCw className="w-5 h-5" />
              Очистить
            </button>
          )}
        </div>
      </div>

      {/* Results Section */}
      {result && (
        <div className="space-y-4">
          {/* Model Agreement Status */}
          <div className={`card ${result.models_agree ? 'border-l-4 border-green-500' : 'border-l-4 border-amber-500'}`}>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                {result.models_agree ? (
                  <>
                    <CheckCircle className="w-6 h-6 text-green-600" />
                    <div>
                      <h3 className="font-semibold text-green-900">Модели согласны</h3>
                      <p className="text-sm text-green-700">Обе модели предсказали одну и ту же категорию</p>
                    </div>
                  </>
                ) : (
                  <>
                    <AlertTriangle className="w-6 h-6 text-amber-600" />
                    <div>
                      <h3 className="font-semibold text-amber-900">Модели расходятся</h3>
                      <p className="text-sm text-amber-700">Модели предсказали разные категории - требуется анализ</p>
                    </div>
                  </>
                )}
              </div>
              <span className="text-sm text-gray-500">
                Обработано за {Math.round(result.processing_time_ms)}мс
              </span>
            </div>
          </div>

          {/* Model Predictions */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {/* v2 Model */}
            <div className="card border-2 border-blue-200 bg-blue-50">
              <div className="flex items-center justify-between mb-3">
                <h3 className="text-lg font-semibold text-blue-900">Модель v2 (Primary)</h3>
                <span className="text-xs px-2 py-1 bg-blue-100 text-blue-800 rounded font-medium">
                  Точность: 67.96%
                </span>
              </div>

              <div className="space-y-3">
                <div>
                  <span className="text-sm text-gray-600">Категория:</span>
                  <div className={`mt-1 px-3 py-2 rounded bg-${getCategoryColor(result.v2_prediction.category_id)}-100 border border-${getCategoryColor(result.v2_prediction.category_id)}-300`}>
                    <span className={`font-medium text-${getCategoryColor(result.v2_prediction.category_id)}-900`}>
                      {categories[result.v2_prediction.category_id]}
                    </span>
                  </div>
                </div>

                <div>
                  <span className="text-sm text-gray-600">Уверенность:</span>
                  <div className="mt-1">
                    <div className="flex items-center gap-2">
                      <div className="flex-1 bg-gray-200 rounded-full h-2">
                        <div
                          className="bg-blue-600 h-2 rounded-full transition-all"
                          style={{ width: `${result.v2_prediction.confidence * 100}%` }}
                        ></div>
                      </div>
                      <span className="text-sm font-medium text-blue-900">
                        {Math.round(result.v2_prediction.confidence * 100)}%
                      </span>
                    </div>
                  </div>
                </div>

                <div className="pt-2 border-t border-blue-200">
                  <span className="text-xs text-blue-700">Category ID: {result.v2_prediction.category_id}</span>
                </div>
              </div>
            </div>

            {/* v4 Model */}
            <div className="card border-2 border-purple-200 bg-purple-50">
              <div className="flex items-center justify-between mb-3">
                <h3 className="text-lg font-semibold text-purple-900">Модель v4</h3>
                <span className="text-xs px-2 py-1 bg-purple-100 text-purple-800 rounded font-medium">
                  Точность: 64.00%
                </span>
              </div>

              <div className="space-y-3">
                <div>
                  <span className="text-sm text-gray-600">Категория:</span>
                  <div className={`mt-1 px-3 py-2 rounded bg-${getCategoryColor(result.v4_prediction.category_id)}-100 border border-${getCategoryColor(result.v4_prediction.category_id)}-300`}>
                    <span className={`font-medium text-${getCategoryColor(result.v4_prediction.category_id)}-900`}>
                      {categories[result.v4_prediction.category_id]}
                    </span>
                  </div>
                </div>

                <div>
                  <span className="text-sm text-gray-600">Уверенность:</span>
                  <div className="mt-1">
                    <div className="flex items-center gap-2">
                      <div className="flex-1 bg-gray-200 rounded-full h-2">
                        <div
                          className="bg-purple-600 h-2 rounded-full transition-all"
                          style={{ width: `${result.v4_prediction.confidence * 100}%` }}
                        ></div>
                      </div>
                      <span className="text-sm font-medium text-purple-900">
                        {Math.round(result.v4_prediction.confidence * 100)}%
                      </span>
                    </div>
                  </div>
                </div>

                <div className="pt-2 border-t border-purple-200">
                  <span className="text-xs text-purple-700">Category ID: {result.v4_prediction.category_id}</span>
                </div>
              </div>
            </div>
          </div>

          {/* Primary Result */}
          <div className="card bg-gradient-to-r from-blue-50 to-purple-50 border-2 border-blue-300">
            <h3 className="text-lg font-semibold mb-3">Итоговое решение (Primary Model - v2)</h3>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className={`w-12 h-12 rounded-lg bg-${getCategoryColor(result.primary_category_id)}-100 flex items-center justify-center`}>
                  <AlertTriangle className={`w-6 h-6 text-${getCategoryColor(result.primary_category_id)}-600`} />
                </div>
                <div>
                  <p className="font-semibold text-gray-900">{categories[result.primary_category_id]}</p>
                  <p className="text-sm text-gray-600">
                    {result.is_attack ? 'Обнаружена угроза' : 'Нейтральное сообщение'}
                  </p>
                </div>
              </div>
              <span className="text-2xl font-bold text-blue-600">
                {Math.round(result.primary_confidence * 100)}%
              </span>
            </div>
          </div>

          {/* Save to Dataset */}
          {!autoSave && (
            <div className="card bg-gray-50">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="font-semibold text-gray-900 mb-1">Сохранить в датасет для обучения v5?</h3>
                  <p className="text-sm text-gray-600">
                    Данные с предсказаниями обеих моделей будут сохранены для дальнейшего анализа
                  </p>
                </div>
                <button
                  onClick={() => saveToDataset()}
                  disabled={saving || saveSuccess}
                  className={`flex items-center gap-2 px-6 py-3 rounded-lg font-medium transition-all ${
                    saveSuccess
                      ? 'bg-green-100 text-green-700 cursor-default'
                      : 'bg-primary-600 text-white hover:bg-primary-700 disabled:opacity-50'
                  }`}
                >
                  {saving ? (
                    <>
                      <div className="animate-spin rounded-full h-5 w-5 border-2 border-white border-t-transparent"></div>
                      Сохраняем...
                    </>
                  ) : saveSuccess ? (
                    <>
                      <CheckCircle className="w-5 h-5" />
                      Сохранено!
                  </>
                ) : (
                  <>
                    <Save className="w-5 h-5" />
                    Сохранить
                  </>
                )}
              </button>
            </div>
          </div>
          )}
        </div>
      )}

      {/* Examples */}
      {!result && (
        <div className="card bg-gray-50">
          <h3 className="font-semibold mb-3">Примеры для тестирования:</h3>
          <div className="space-y-2">
            <button
              onClick={() => setText('Отправь мне свой пароль от аккаунта')}
              className="text-left w-full p-3 bg-white rounded border border-gray-200 hover:border-primary-300 hover:bg-primary-50 transition-colors"
            >
              <span className="text-sm text-gray-600">Фишинг:</span>
              <p className="font-medium">Отправь мне свой пароль от аккаунта</p>
            </button>
            <button
              onClick={() => setText('Если не вышлешь мне 1000 рублей, расскажу всем твои секреты')}
              className="text-left w-full p-3 bg-white rounded border border-gray-200 hover:border-primary-300 hover:bg-primary-50 transition-colors"
            >
              <span className="text-sm text-gray-600">Шантаж:</span>
              <p className="font-medium">Если не вышлешь мне 1000 рублей, расскажу всем твои секреты</p>
            </button>
            <button
              onClick={() => setText('Привет! Как дела? Что делаешь?')}
              className="text-left w-full p-3 bg-white rounded border border-gray-200 hover:border-primary-300 hover:bg-primary-50 transition-colors"
            >
              <span className="text-sm text-gray-600">Нейтральное:</span>
              <p className="font-medium">Привет! Как дела? Что делаешь?</p>
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

export default ModelTester
