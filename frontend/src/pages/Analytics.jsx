import { useEffect, useState } from 'react'
import { analyticsService } from '../services/api'
import { TrendingUp, AlertTriangle, Shield, Activity } from 'lucide-react'
import { BarChart, Bar, LineChart, Line, PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'

const Analytics = () => {
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchAnalytics()
  }, [])

  const fetchAnalytics = async () => {
    try {
      setLoading(true)
      const analyticsData = await analyticsService.getOverview()
      setData(analyticsData)
    } catch (error) {
      console.error('Failed to fetch analytics:', error)
    } finally {
      setLoading(false)
    }
  }

  const getCategoryName = (categoryId) => {
    const categories = {
      1: 'Груминг',
      2: 'Шантаж',
      3: 'Буллинг',
      4: 'Суицид',
      5: 'Опасные игры',
      6: 'Пропаганда',
      7: 'Мошенничество',
      8: 'Фишинг',
      9: 'Нейтральное',
    }
    return categories[categoryId] || 'Неизвестно'
  }

  const COLORS = ['#ef4444', '#f97316', '#f59e0b', '#eab308', '#84cc16', '#22c55e', '#10b981', '#14b8a6', '#6b7280']

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
      </div>
    )
  }

  const categoryData = data?.category_distribution
    ? Object.entries(data.category_distribution).map(([categoryId, count]) => ({
        name: getCategoryName(parseInt(categoryId)),
        value: count,
        categoryId: parseInt(categoryId),
      }))
    : []

  const trendData = data?.daily_trends || []

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-gray-900">Аналитика</h1>
        <p className="text-gray-500 mt-1">Статистика и тренды обнаружения угроз</p>
      </div>

      {/* Key Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <div className="card">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">Всего инцидентов</p>
              <p className="text-3xl font-bold text-gray-900 mt-2">
                {data?.total_incidents?.toLocaleString() || 0}
              </p>
            </div>
            <div className="w-12 h-12 rounded-lg bg-danger-100 flex items-center justify-center">
              <AlertTriangle className="w-6 h-6 text-danger-600" />
            </div>
          </div>
        </div>

        <div className="card">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">Проверено сообщений</p>
              <p className="text-3xl font-bold text-gray-900 mt-2">
                {data?.total_messages?.toLocaleString() || 0}
              </p>
            </div>
            <div className="w-12 h-12 rounded-lg bg-blue-100 flex items-center justify-center">
              <Activity className="w-6 h-6 text-blue-600" />
            </div>
          </div>
        </div>

        <div className="card">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">Процент угроз</p>
              <p className="text-3xl font-bold text-gray-900 mt-2">
                {data?.detection_rate
                  ? `${(data.detection_rate * 100).toFixed(1)}%`
                  : 'Н/Д'}
              </p>
              <p className="text-xs text-gray-500 mt-1">от всех сообщений</p>
            </div>
            <div className="w-12 h-12 rounded-lg bg-green-100 flex items-center justify-center">
              <Shield className="w-6 h-6 text-green-600" />
            </div>
          </div>
        </div>

        <div className="card">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">За 24 часа</p>
              <p className="text-3xl font-bold text-gray-900 mt-2">
                {data?.incidents_24h?.toLocaleString() || 0}
              </p>
            </div>
            <div className="w-12 h-12 rounded-lg bg-yellow-100 flex items-center justify-center">
              <TrendingUp className="w-6 h-6 text-yellow-600" />
            </div>
          </div>
        </div>
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Category Distribution */}
        <div className="card">
          <h2 className="text-xl font-semibold text-gray-900 mb-6">
            Распределение по категориям
          </h2>
          {categoryData.length > 0 ? (
            <ResponsiveContainer width="100%" height={300}>
              <PieChart>
                <Pie
                  data={categoryData}
                  cx="50%"
                  cy="50%"
                  labelLine={false}
                  label={(entry) => `${entry.name}: ${entry.value}`}
                  outerRadius={100}
                  fill="#8884d8"
                  dataKey="value"
                >
                  {categoryData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          ) : (
            <div className="h-64 flex items-center justify-center text-gray-500">
              Нет данных для отображения
            </div>
          )}
        </div>

        {/* Category Bar Chart */}
        <div className="card">
          <h2 className="text-xl font-semibold text-gray-900 mb-6">
            Количество инцидентов по типам
          </h2>
          {categoryData.length > 0 ? (
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={categoryData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="name" angle={-45} textAnchor="end" height={100} />
                <YAxis />
                <Tooltip />
                <Bar dataKey="value" fill="#0ea5e9" />
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div className="h-64 flex items-center justify-center text-gray-500">
              Нет данных для отображения
            </div>
          )}
        </div>
      </div>

      {/* Trends */}
      {trendData.length > 0 && (
        <div className="card">
          <h2 className="text-xl font-semibold text-gray-900 mb-6">
            Тренд обнаружения угроз
          </h2>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={trendData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="date" />
              <YAxis />
              <Tooltip />
              <Legend />
              <Line type="monotone" dataKey="incidents" stroke="#ef4444" name="Инциденты" />
              <Line type="monotone" dataKey="messages" stroke="#0ea5e9" name="Сообщения" />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}

      {/* Category Details Table */}
      <div className="card">
        <h2 className="text-xl font-semibold text-gray-900 mb-6">
          Детальная статистика по категориям
        </h2>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Категория
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Количество
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Процент
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {categoryData.map((item, index) => {
                const total = categoryData.reduce((sum, cat) => sum + cat.value, 0)
                const percentage = total > 0 ? (item.value / total) * 100 : 0

                return (
                  <tr key={index} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center">
                        <div
                          className="w-3 h-3 rounded-full mr-3"
                          style={{ backgroundColor: COLORS[index % COLORS.length] }}
                        />
                        <span className="text-sm font-medium text-gray-900">{item.name}</span>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm text-gray-900">
                      {item.value.toLocaleString()}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm text-gray-500">
                      {percentage.toFixed(1)}%
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}

export default Analytics
