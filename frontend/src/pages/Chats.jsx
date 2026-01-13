import { useEffect, useState } from 'react'
import { chatService } from '../services/api'
import { MessageSquare, Users, CheckCircle, XCircle, Send, MessageCircle } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { ru } from 'date-fns/locale'

const Chats = () => {
  const [chats, setChats] = useState([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchChats()
  }, [])

  const fetchChats = async () => {
    try {
      setLoading(true)
      const data = await chatService.getAll()
      setChats(data.chats || [])
    } catch (error) {
      console.error('Failed to fetch chats:', error)
    } finally {
      setLoading(false)
    }
  }

  const getChatTypeIcon = (chatType) => {
    return chatType === 'group' || chatType === 'supergroup' || chatType === 'chat' ? Users : MessageSquare
  }

  const getChatTypeName = (chatType) => {
    const types = {
      'private': 'Личный чат',
      'user': 'Личный чат',
      'group': 'Группа',
      'supergroup': 'Супергруппа',
      'channel': 'Канал',
      'chat': 'Беседа',
    }
    return types[chatType] || 'Неизвестно'
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

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Чаты</h1>
          <p className="text-gray-500 mt-1">Мониторимые чаты из Telegram и VK</p>
        </div>
      </div>

      {/* Chats Grid */}
      {loading ? (
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
        </div>
      ) : chats.length === 0 ? (
        <div className="card text-center py-12">
          <MessageSquare className="w-12 h-12 text-gray-400 mx-auto mb-3" />
          <p className="text-gray-500">Чаты не найдены</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {chats.map((chat) => {
            const Icon = getChatTypeIcon(chat.chat_type)
            const SourceIcon = getSourceIcon(chat.source)
            return (
              <div key={chat.id} className="card hover:shadow-md transition-shadow">
                <div className="flex items-start gap-4">
                  <div className="w-12 h-12 bg-primary-100 rounded-lg flex items-center justify-center flex-shrink-0">
                    <Icon className="w-6 h-6 text-primary-600" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <h3 className="font-semibold text-gray-900 truncate">
                        {chat.title || 'Без названия'}
                      </h3>
                      <div className={`${getSourceColor(chat.source)} text-white text-xs px-2 py-0.5 rounded flex items-center gap-1 flex-shrink-0`}>
                        <SourceIcon className="w-3 h-3" />
                        {getSourceName(chat.source)}
                      </div>
                    </div>
                    <p className="text-sm text-gray-500 mt-1">
                      {getChatTypeName(chat.chat_type)}
                    </p>
                  </div>
                  {chat.is_monitored ? (
                    <CheckCircle className="w-5 h-5 text-green-500 flex-shrink-0" />
                  ) : (
                    <XCircle className="w-5 h-5 text-gray-400 flex-shrink-0" />
                  )}
                </div>

                <div className="mt-4 pt-4 border-t border-gray-200 space-y-2">
                  <div className="flex justify-between text-sm">
                    <span className="text-gray-600">Участников:</span>
                    <span className="font-medium text-gray-900">
                      {chat.member_count?.toLocaleString() || 'Н/Д'}
                    </span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-gray-600">Сообщений собрано:</span>
                    <span className="font-medium text-gray-900">
                      {chat.message_count?.toLocaleString() || 0}
                    </span>
                  </div>
                  {chat.last_message_date && (
                    <div className="flex justify-between text-sm">
                      <span className="text-gray-600">Последнее сообщение:</span>
                      <span className="font-medium text-gray-900">
                        {formatDistanceToNow(new Date(chat.last_message_date), {
                          addSuffix: true,
                          locale: ru,
                        })}
                      </span>
                    </div>
                  )}
                </div>

                <div className="mt-4 flex gap-2 flex-wrap">
                  <span className={`badge ${chat.is_monitored ? 'badge-success' : 'badge-info'}`}>
                    {chat.is_monitored ? 'Мониторится' : 'Не активен'}
                  </span>
                  {chat.telegram_id && (
                    <span className="badge bg-gray-100 text-gray-700 text-xs">
                      TG ID: {chat.telegram_id}
                    </span>
                  )}
                  {chat.vk_peer_id && (
                    <span className="badge bg-gray-100 text-gray-700 text-xs">
                      VK ID: {chat.vk_peer_id}
                    </span>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

export default Chats
