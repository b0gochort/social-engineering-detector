import { useEffect, useState } from 'react'
import { incidentService } from '../services/api'
import { AlertTriangle, Search, Filter, ChevronLeft, ChevronRight, Lock, Unlock, Send, MessageCircle } from 'lucide-react'
import { format } from 'date-fns'
import { ru } from 'date-fns/locale'

const Incidents = () => {
  const [incidents, setIncidents] = useState([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [categoryFilter, setCategoryFilter] = useState('all')
  const [page, setPage] = useState(1)
  const [totalPages, setTotalPages] = useState(1)
  const [requestingAccess, setRequestingAccess] = useState({})

  useEffect(() => {
    fetchIncidents()
  }, [page, categoryFilter])

  const fetchIncidents = async () => {
    try {
      setLoading(true)
      const params = {
        page,
        limit: 20,
      }
      if (categoryFilter !== 'all') {
        params.category_id = categoryFilter
      }
      const data = await incidentService.getAll(params)
      setIncidents(data.incidents || [])
      setTotalPages(data.total_pages || 1)
    } catch (error) {
      console.error('Failed to fetch incidents:', error)
    } finally {
      setLoading(false)
    }
  }

  const requestAccess = async (incidentId) => {
    try {
      setRequestingAccess(prev => ({ ...prev, [incidentId]: true }))
      const response = await fetch(`http://localhost:8080/api/access-requests`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        },
        body: JSON.stringify({ incident_id: incidentId })
      })

      if (response.ok) {
        // Refresh incidents to show updated status
        await fetchIncidents()
      } else {
        const error = await response.json()
        alert(error.error || 'Не удалось отправить запрос на доступ')
      }
    } catch (error) {
      console.error('Failed to request access:', error)
      alert('Не удалось отправить запрос на доступ')
    } finally {
      setRequestingAccess(prev => ({ ...prev, [incidentId]: false }))
    }
  }

  const getCategoryName = (categoryId) => {
    const categories = {
      1: 'Груминг',
      2: 'Шантаж',
      3: 'Буллинг',
      4: 'Склонение к суициду',
      5: 'Опасные игры',
      6: 'Пропаганда веществ',
      7: 'Финансовое мошенничество',
      8: 'Фишинг',
    }
    return categories[categoryId] || 'Неизвестно'
  }

  const getCategoryColor = (categoryId) => {
    const colors = {
      1: 'danger',
      2: 'danger',
      3: 'warning',
      4: 'danger',
      5: 'danger',
      6: 'warning',
      7: 'warning',
      8: 'warning',
    }
    return colors[categoryId] || 'info'
  }

  const getSourceName = (source) => {
    return source === 'vk' ? 'VK' : 'Telegram'
  }

  const getSourceColor = (source) => {
    return source === 'vk' ? 'bg-blue-500' : 'bg-sky-500'
  }

  const getSourceIcon = (source) => {
    return source === 'vk' ? MessageCircle : Send
  }

  const filteredIncidents = incidents.filter((incident) =>
    incident.message_text?.toLowerCase().includes(search.toLowerCase())
  )

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-gray-900">Инциденты</h1>
        <p className="text-gray-500 mt-1">Обнаруженные угрозы и подозрительные сообщения</p>
      </div>

      {/* Filters */}
      <div className="card">
        <div className="flex flex-col md:flex-row gap-4">
          {/* Search */}
          <div className="flex-1 relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
            <input
              type="text"
              placeholder="Поиск по тексту сообщения..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="input pl-10"
            />
          </div>

          {/* Category Filter */}
          <div className="relative">
            <Filter className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
            <select
              value={categoryFilter}
              onChange={(e) => {
                setCategoryFilter(e.target.value)
                setPage(1)
              }}
              className="input pl-10 pr-10"
            >
              <option value="all">Все категории</option>
              <option value="1">Груминг</option>
              <option value="2">Шантаж</option>
              <option value="3">Буллинг</option>
              <option value="4">Склонение к суициду</option>
              <option value="5">Опасные игры</option>
              <option value="6">Пропаганда веществ</option>
              <option value="7">Финансовое мошенничество</option>
              <option value="8">Фишинг</option>
            </select>
          </div>
        </div>
      </div>

      {/* Incidents List */}
      <div className="card">
        {loading ? (
          <div className="flex items-center justify-center h-64">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
          </div>
        ) : filteredIncidents.length === 0 ? (
          <div className="text-center py-12">
            <AlertTriangle className="w-12 h-12 text-gray-400 mx-auto mb-3" />
            <p className="text-gray-500">Инцидентов не найдено</p>
          </div>
        ) : (
          <div className="space-y-4">
            {filteredIncidents.map((incident) => (
              <div
                key={incident.id}
                className="border border-gray-200 rounded-lg p-5 hover:border-gray-300 transition-colors"
              >
                <div className="flex items-start justify-between mb-3">
                  <div className="flex items-center gap-3">
                    <div className={`w-10 h-10 rounded-lg bg-${getCategoryColor(incident.category_id)}-100 flex items-center justify-center flex-shrink-0`}>
                      <AlertTriangle className={`w-5 h-5 text-${getCategoryColor(incident.category_id)}-600`} />
                    </div>
                    <div>
                      <span className={`badge-danger`}>
                        {incident.threat_type || 'Неизвестно'}
                      </span>
                      {incident.confidence && (
                        <span className="ml-2 text-xs text-gray-500">
                          Уверенность: {Math.round(incident.confidence * 100)}%
                        </span>
                      )}

                      {/* Dual model predictions */}
                      {(incident.v2_category_id || incident.v4_category_id) && (
                        <div className="flex items-center gap-2 mt-2">
                          {incident.v2_category_id && (
                            <span className="text-xs px-2 py-1 bg-blue-50 text-blue-700 rounded border border-blue-200">
                              v2: {getCategoryName(incident.v2_category_id)}
                            </span>
                          )}
                          {incident.v4_category_id && (
                            <span className="text-xs px-2 py-1 bg-purple-50 text-purple-700 rounded border border-purple-200">
                              v4: {getCategoryName(incident.v4_category_id)}
                            </span>
                          )}
                          {incident.models_agree !== null && (
                            incident.models_agree ? (
                              <span className="text-xs px-2 py-1 bg-green-50 text-green-700 rounded border border-green-200">
                                ✓ Согласны
                              </span>
                            ) : (
                              <span className="text-xs px-2 py-1 bg-amber-50 text-amber-700 rounded border border-amber-200">
                                ⚠ Расходятся
                              </span>
                            )
                          )}
                        </div>
                      )}
                    </div>
                  </div>
                  <span className="text-xs text-gray-500">
                    {format(new Date(incident.created_at), 'dd MMM yyyy, HH:mm', { locale: ru })}
                  </span>
                </div>

                <div className="bg-gray-50 rounded-lg p-4 mb-3">
                  {console.log('Incident text:', incident.message_text, 'Access granted:', incident.access_granted, 'Request ID:', incident.current_access_request_id)}
                  {incident.message_text && incident.message_text.includes('[Для просмотра текста запросите доступ]') ? (
                    <div className="space-y-3">
                      <div className="flex items-center gap-2 text-gray-600">
                        <Lock className="w-5 h-5" />
                        <p className="font-medium">{incident.message_text}</p>
                      </div>
                      {incident.current_access_request_id ? (
                        <div className="flex items-center gap-2 text-blue-600 text-sm">
                          <div className="animate-pulse">⏳</div>
                          <span className="font-medium">Запрос отправлен, ожидайте ответа</span>
                        </div>
                      ) : (
                        <button
                          onClick={() => requestAccess(incident.id)}
                          disabled={requestingAccess[incident.id]}
                          className="btn-primary text-sm flex items-center gap-2 disabled:opacity-50"
                        >
                          {requestingAccess[incident.id] ? (
                            <>
                              <div className="animate-spin rounded-full h-4 w-4 border-2 border-white border-t-transparent"></div>
                              Отправка запроса...
                            </>
                          ) : (
                            <>
                              <Unlock className="w-4 h-4" />
                              Запросить доступ
                            </>
                          )}
                        </button>
                      )}
                    </div>
                  ) : incident.access_granted ? (
                    <div className="space-y-2">
                      <div className="flex items-center gap-2 text-green-600 text-sm mb-2">
                        <Unlock className="w-4 h-4" />
                        <span className="font-medium">Доступ предоставлен</span>
                      </div>
                      <p className="text-gray-800 whitespace-pre-wrap">{incident.message_text}</p>
                    </div>
                  ) : (
                    <p className="text-gray-800 whitespace-pre-wrap">{incident.message_text}</p>
                  )}
                </div>

                <div className="flex items-center gap-4 text-sm text-gray-500 flex-wrap">
                  <div className="flex items-center gap-2">
                    <span>Чат: <span className="font-medium text-gray-700">{incident.chat_title || 'Неизвестно'}</span></span>
                    {incident.source && (() => {
                      const SourceIcon = getSourceIcon(incident.source)
                      return (
                        <div className={`${getSourceColor(incident.source)} text-white text-xs px-2 py-0.5 rounded flex items-center gap-1`}>
                          <SourceIcon className="w-3 h-3" />
                          {getSourceName(incident.source)}
                        </div>
                      )
                    })()}
                  </div>
                  <span>•</span>
                  <span>ID: #{incident.id}</span>
                  {incident.sender_name && (
                    <>
                      <span>•</span>
                      <span>От: <span className="font-medium text-gray-700">{incident.sender_name}</span></span>
                    </>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between mt-6 pt-6 border-t border-gray-200">
            <div className="text-sm text-gray-500">
              Страница {page} из {totalPages}
            </div>
            <div className="flex gap-2">
              <button
                onClick={() => setPage(p => Math.max(1, p - 1))}
                disabled={page === 1}
                className="btn-secondary disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1"
              >
                <ChevronLeft className="w-4 h-4" />
                Назад
              </button>
              <button
                onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                disabled={page === totalPages}
                className="btn-secondary disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1"
              >
                Вперёд
                <ChevronRight className="w-4 h-4" />
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

export default Incidents
