export default {
  app: {
    title: 'TelegramPV',
    version: '1.0.0',
    subtitle: 'Telegram Chat & Channel Manager'
  },
  nav: {
    allChats: 'Groups & Channels',
    groups: 'Groups',
    channels: 'Channels',
    bots: 'Bots',
    users: 'Personal Chats',
    settings: 'Settings',
    logs: 'System Logs'
  },
  auth: {
    title: 'Telegram Authentication',
    step1Title: 'API Credentials',
    step1Desc: 'Enter your Telegram API ID and API Hash from my.telegram.org',
    step2Title: 'Phone Number',
    step2Desc:
      'Enter your phone number registered with Telegram including country code (e.g. +1234567890)',
    step3Title: 'Verification Code',
    step3Desc:
      'Enter the 5-digit verification code sent to your Telegram account',
    step4Title: 'Two-Factor Authentication (2FA)',
    step4Desc: 'Enter your 2FA password to unlock your account',
    apiIdLabel: 'API ID',
    apiHashLabel: 'API Hash',
    phoneLabel: 'Phone Number',
    codeLabel: 'Telegram Code',
    passwordLabel: '2FA Password',
    submit: 'Continue',
    back: 'Back',
    sendCode: 'Send Code',
    verifyCode: 'Verify Code',
    unlock2FA: 'Unlock Account',
    credsBannerA: 'You need an API ID and API Hash from ',
    credsBannerB: 'my.telegram.org',
    errFillBoth: 'Please fill in both API ID and API Hash',
    errApiIdFormat: 'API ID must contain only digits',
    errApiHashFormat: 'API Hash must be a 32-character hexadecimal string',
    errNoResponse: 'No response from server',
    errInvalidResponse: 'Invalid response from server',
    errProcess: 'Authentication failed'
  },
  chats: {
    searchPlaceholder: 'Search chats by name, username or ID...',
    syncButton: 'Refresh from Telegram',
    syncing: 'Refreshing...',
    exportButton: 'Export TXT',
    deleteSelected: 'Delete Selected (%d)',
    selectAll: 'Select All',
    deselectAll: 'Deselect All',
    emptyTitle: 'No chats found',
    emptyDesc:
      'Click "Refresh from Telegram" to fetch your chats or try a different search filter',
    confirmDeleteTitle: 'Confirm Chat Removal',
    confirmDeleteMsg:
      'Are you sure you want to leave or remove "%s"? This action cannot be undone.',
    confirmBatchDeleteTitle: 'Confirm Batch Chat Removal',
    confirmBatchDeleteMsg:
      'Are you sure you want to leave or remove %d selected chats? This action cannot be undone.',
    leaveSuccess: 'Successfully left/deleted chat',
    batchLeaveSuccess: 'Successfully processed batch chat removal',
    id: 'ID',
    title: 'Name',
    type: 'Type',
    username: 'Username',
    unread: 'Unread',
    actions: 'Actions',
    group: 'Group',
    channel: 'Channel',
    bot: 'Bot',
    user: 'User',
    leaveAction: 'Leave / Delete'
  },
  settings: {
    title: 'Application Settings',
    appearance: 'Theme & Appearance',
    themeSelect: 'Select Theme',
    language: 'Language',
    languageSelect: 'Select Language',
    session: 'Telegram Session',
    connectedAs: 'Connected as',
    phone: 'Phone',
    logout: 'Log Out Session',
    logoutConfirm:
      'Are you sure you want to log out? You will need to re-authenticate.',
    saveSuccess: 'Settings saved successfully'
  },
  logs: {
    title: 'System Activity Logs',
    clear: 'Clear View',
    empty: 'No log entries available'
  },
  common: {
    confirm: 'Confirm',
    cancel: 'Cancel',
    close: 'Close',
    error: 'Error',
    success: 'Success',
    loading: 'Loading...'
  }
}
