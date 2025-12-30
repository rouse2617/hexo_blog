import Dexie, { Table } from 'dexie'

export interface Message {
  id?: number
  sessionId: number
  role: 'user' | 'assistant' | 'system'
  content: string
  timestamp: Date
  metadata?: Record<string, any>
}

export interface Session {
  id?: number
  title: string
  createdAt: Date
  updatedAt: Date
  messageCount: number
}

export interface Draft {
  id?: number
  sessionId: number
  content: string
  timestamp: Date
}

class ChatDatabase extends Dexie {
  messages!: Table<Message>
  sessions!: Table<Session>
  drafts!: Table<Draft>

  constructor() {
    super('AIOPsChatDB')

    this.version(1).stores({
      messages: '++id, sessionId, timestamp',
      sessions: '++id, createdAt, updatedAt',
      drafts: '++id, sessionId, timestamp'
    })
  }

  async saveMessages(sessionId: number, messages: Message[]): Promise<void> {
    await this.transaction('rw', this.messages, async () => {
      await this.messages.where('sessionId').equals(sessionId).delete()
      await this.messages.bulkAdd(messages)
    })
  }

  async getMessages(sessionId: number): Promise<Message[]> {
    return await this.messages
      .where('sessionId')
      .equals(sessionId)
      .sortBy('timestamp')
  }

  async createSession(title: string): Promise<number> {
    const now = new Date()
    return await this.sessions.add({
      title,
      createdAt: now,
      updatedAt: now,
      messageCount: 0
    })
  }

  async updateSessionTitle(id: number, title: string): Promise<void> {
    await this.sessions.update(id, {
      title,
      updatedAt: new Date()
    })
  }

  async updateSessionMessageCount(id: number, count: number): Promise<void> {
    await this.sessions.update(id, {
      messageCount: count,
      updatedAt: new Date()
    })
  }

  async getSessions(): Promise<Session[]> {
    return await this.sessions
      .orderBy('updatedAt')
      .reverse()
      .toArray()
  }

  async deleteSession(id: number): Promise<void> {
    await this.transaction('rw', [this.sessions, this.messages, this.drafts], async () => {
      await this.messages.where('sessionId').equals(id).delete()
      await this.drafts.where('sessionId').equals(id).delete()
      await this.sessions.delete(id)
    })
  }

  async saveDraft(sessionId: number, content: string): Promise<void> {
    const now = new Date()
    const existingDraft = await this.drafts.where('sessionId').equals(sessionId).first()

    if (existingDraft) {
      await this.drafts.update(existingDraft.id!, { content, timestamp: now })
    } else {
      await this.drafts.add({ sessionId, content, timestamp: now })
    }
  }

  async getDraft(sessionId: number): Promise<Draft | undefined> {
    return await this.drafts.where('sessionId').equals(sessionId).first()
  }

  async exportToMarkdown(sessionId: number): Promise<string> {
    const session = await this.sessions.get(sessionId)
    const messages = await this.getMessages(sessionId)

    let markdown = `# ${session?.title || 'Chat Export'}\n\n`
    markdown += `**Created:** ${session?.createdAt.toLocaleString()}\n`
    markdown += `**Messages:** ${session?.messageCount || 0}\n\n`
    markdown += `---\n\n`

    for (const message of messages) {
      const role = message.role === 'user' ? '👤 User' : '🤖 Assistant'
      markdown += `## ${role}\n\n`
      markdown += `${message.content}\n\n`
      markdown += `*${message.timestamp.toLocaleString()}*\n\n`
      markdown += `---\n\n`
    }

    return markdown
  }

  async importFromMarkdown(content: string): Promise<number> {
    // Parse markdown to extract session title and messages
    const lines = content.split('\n')
    let title = 'Imported Chat'
    const messages: Message[] = []
    let currentRole: 'user' | 'assistant' | 'system' = 'user'
    let currentContent: string[] = []

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i]

      if (line.startsWith('# ')) {
        title = line.substring(2).trim()
      } else if (line.startsWith('## 👤 User')) {
        if (currentContent.length > 0) {
          messages.push({
            sessionId: 0, // Will be replaced with actual session ID
            role: currentRole,
            content: currentContent.join('\n').trim(),
            timestamp: new Date()
          })
        }
        currentRole = 'user'
        currentContent = []
      } else if (line.startsWith('## 🤖 Assistant')) {
        if (currentContent.length > 0) {
          messages.push({
            sessionId: 0,
            role: currentRole,
            content: currentContent.join('\n').trim(),
            timestamp: new Date()
          })
        }
        currentRole = 'assistant'
        currentContent = []
      } else if (!line.startsWith('---') && !line.startsWith('*') && !line.startsWith('**')) {
        currentContent.push(line)
      }
    }

    if (currentContent.length > 0) {
      messages.push({
        sessionId: 0,
        role: currentRole,
        content: currentContent.join('\n').trim(),
        timestamp: new Date()
      })
    }

    // Create session and save messages
    const sessionId = await this.createSession(title)
    messages.forEach(msg => msg.sessionId = sessionId)
    await this.messages.bulkAdd(messages)
    await this.updateSessionMessageCount(sessionId, messages.length)

    return sessionId
  }
}

export const chatDB = new ChatDatabase()
