import { useEffect, useState } from 'react'
import { incidentService, analyticsService } from '../services/api'
import { AlertTriangle, MessageSquare, Shield, TrendingUp } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { ru } from 'date-fns/locale'

const Dashboard = () => {
  const [stats, setStats] = useState(null)
  const [recentIncidents, setRecentIncidents] = useState([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchData()
  }, [])

  const fetchData = async () => {
    try {
      setLoading(true)
      const [statsData, incidentsData] = await Promise.all([
        analyticsService.getOverview(),
        incidentService.getAll({ limit: 5, sort: '-created_at' })
      ])
      setStats(statsData)
      setRecentIncidents(incidentsData.incidents || [])
    } catch (error) {
      console.error('Failed to fetch dashboard data:', error)
    } finally {
      setLoading(false)
    }
  }

  const getCategoryColor = (threatType) => {
    if (!threatType) return 'info'
    // Определяем цвет на основе строкового типа угрозы
    const type = threatType.toLowerCase()
    if (type.includes('груминг') || type.includes('сексуальн') ||
        type.includes('угроз') || type.includes('шантаж') || type.includes('вымогательство') ||
        type.includes('суицид') || type.includes('самоповреждени') ||
        type.includes('насилие') || type.includes('буллинг')) {
      return 'danger'
    }
    return 'warning'
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
      </div>
    )
  }

  const statCards = [
    {
      title: 'Всего инцидентов',
      value: stats?.total_incidents || 0,
      icon: AlertTriangle,
      color: 'danger',
      trend: stats?.incidents_trend || 0,
    },
    {
      title: 'Сообщений проверено',
      value: stats?.total_messages || 0,
      icon: MessageSquare,
      color: 'info',
      trend: stats?.messages_trend || 0,
    },
    {
      title: 'Активных чатов',
      value: stats?.active_chats || 0,
      icon: Shield,
      color: 'success',
      trend: 0,
    },
    {
      title: 'За последние 24ч',
      value: stats?.incidents_24h || 0,
      icon: TrendingUp,
      color: 'warning',
      trend: stats?.incidents_24h_trend || 0,
    },
  ]

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-gray-900">Дашборд</h1>
        <p className="text-gray-500 mt-1">Обзор системы мониторинга угроз</p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {statCards.map((stat, index) => (
          <div key={index} className="card hover:shadow-md transition-shadow">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600">{stat.title}</p>
                <p className="text-3xl font-bold text-gray-900 mt-2">
                  {stat.value.toLocaleString()}
                </p>
                {stat.trend !== 0 && (
                  <p className={`text-sm mt-1 ${stat.trend > 0 ? 'text-green-600' : 'text-red-600'}`}>
                    {stat.trend > 0 ? '+' : ''}{stat.trend}% vs предыдущий период
                  </p>
                )}
              </div>
              <div className={`w-12 h-12 rounded-lg bg-${stat.color}-100 flex items-center justify-center`}>
                <stat.icon className={`w-6 h-6 text-${stat.color}-600`} />
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Recent Incidents */}
      <div className="card">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-xl font-semibold text-gray-900">Последние инциденты</h2>
          <a href="/incidents" className="text-sm text-primary-600 hover:text-primary-700 font-medium">
            Показать все →
          </a>
        </div>

        {recentIncidents.length === 0 ? (
          <div className="text-center py-12">
            <Shield className="w-12 h-12 text-gray-400 mx-auto mb-3" />
            <p className="text-gray-500">Инцидентов не обнаружено</p>
          </div>
        ) : (
          <div className="space-y-4">
            {recentIncidents.map((incident) => (
              <div
                key={incident.id}
                className="border border-gray-200 rounded-lg p-4 hover:border-gray-300 transition-colors"
              >
                <div className="flex items-start justify-between mb-2">
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-1">
                      <span className={`badge-${getCategoryColor(incident.threat_type)}`}>
                        {incident.threat_type || 'Неизвестно'}
                      </span>
                      {incident.confidence && (
                        <span className="text-xs text-gray-500">
                          {Math.round(incident.confidence * 100)}% уверенность
                        </span>
                      )}
                    </div>
                    <p className="text-gray-700 line-clamp-2">{incident.message_text}</p>
                  </div>
                </div>
                <div className="flex items-center gap-4 text-xs text-gray-500 mt-3">
                  <span>Чат: {incident.chat_title || 'Неизвестно'}</span>
                  <span>•</span>
                  <span>
                    {formatDistanceToNow(new Date(incident.created_at), {
                      addSuffix: true,
                      locale: ru,
                    })}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

    </div>
  )
}

export default Dashboard
