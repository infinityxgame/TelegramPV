export default {
  app: {
    title: 'TelegramPV',
    version: '1.0.0',
    subtitle: 'Administrador de Chats y Canales de Telegram'
  },
  nav: {
    allChats: 'Grupos y Canales',
    groups: 'Grupos',
    channels: 'Canales',
    bots: 'Bots',
    users: 'Chats Personales',
    settings: 'Ajustes',
    logs: 'Registros del Sistema'
  },
  auth: {
    title: 'Autenticación de Telegram',
    step1Title: 'Credenciales de API',
    step1Desc: 'Introduce tu API ID y API Hash obtenidos de my.telegram.org',
    step2Title: 'Número de Teléfono',
    step2Desc:
      'Introduce tu número de teléfono registrado en Telegram con código de país (ej. +34600000000)',
    step3Title: 'Código de Verificación',
    step3Desc: 'Introduce el código de 5 dígitos enviado a tu Telegram',
    step4Title: 'Verificación en Dos Pasos (2FA)',
    step4Desc: 'Introduce tu contraseña 2FA para desbloquear tu cuenta',
    apiIdLabel: 'API ID',
    apiHashLabel: 'API Hash',
    phoneLabel: 'Número de Teléfono',
    codeLabel: 'Código de Telegram',
    passwordLabel: 'Contraseña 2FA',
    submit: 'Continuar',
    back: 'Atrás',
    sendCode: 'Enviar Código',
    verifyCode: 'Verificar Código',
    unlock2FA: 'Desbloquear Cuenta',
    credsBannerA: 'Necesitas un API ID y API Hash desde ',
    credsBannerB: 'my.telegram.org',
    errFillBoth: 'Por favor rellena tanto el API ID como el API Hash',
    errApiIdFormat: 'El API ID debe contener solo números',
    errApiHashFormat: 'El API Hash debe tener 32 caracteres hexadecimales',
    errNoResponse: 'Sin respuesta del servidor',
    errInvalidResponse: 'Respuesta no válida del servidor',
    errProcess: 'Error en la autenticación'
  },
  chats: {
    searchPlaceholder: 'Buscar chats por nombre, @usuario o ID...',
    syncButton: 'Actualizar desde Telegram',
    syncing: 'Actualizando...',
    exportButton: 'Exportar TXT',
    deleteSelected: 'Eliminar Seleccionados (%d)',
    selectAll: 'Seleccionar Todos',
    deselectAll: 'Deseleccionar Todos',
    emptyTitle: 'No se encontraron chats',
    emptyDesc:
      'Haz clic en "Actualizar desde Telegram" para consultar tus chats o usa otro filtro de búsqueda',
    confirmDeleteTitle: 'Confirmar eliminación del chat',
    confirmDeleteMsg:
      '¿Estás seguro de que deseas salir o eliminar "%s"? Esta acción no se puede deshacer.',
    confirmBatchDeleteTitle: 'Confirmar eliminación masiva',
    confirmBatchDeleteMsg:
      '¿Estás seguro de que deseas salir o eliminar los %d chats seleccionados? Esta acción no se puede deshacer.',
    leaveSuccess: 'Chat eliminado / abandonado con éxito',
    batchLeaveSuccess: 'Procesada la eliminación masiva de chats con éxito',
    id: 'ID',
    title: 'Nombre',
    type: 'Tipo',
    username: 'Usuario',
    unread: 'No leídos',
    actions: 'Acciones',
    group: 'Grupo',
    channel: 'Canal',
    bot: 'Bot',
    user: 'Usuario',
    leaveAction: 'Salir / Eliminar'
  },
  settings: {
    title: 'Ajustes de la Aplicación',
    appearance: 'Tema y Apariencia',
    themeSelect: 'Seleccionar Tema',
    language: 'Idioma',
    languageSelect: 'Seleccionar Idioma',
    session: 'Sesión de Telegram',
    connectedAs: 'Conectado como',
    phone: 'Teléfono',
    logout: 'Cerrar Sesión',
    logoutConfirm:
      '¿Estás seguro de que deseas cerrar sesión? Tendrás que volver a autenticarte.',
    saveSuccess: 'Ajustes guardados correctamente'
  },
  logs: {
    title: 'Registros de Actividad',
    clear: 'Limpiar Vista',
    empty: 'No hay entradas de registro disponibles'
  },
  common: {
    confirm: 'Confirmar',
    cancel: 'Cancelar',
    close: 'Cerrar',
    error: 'Error',
    success: 'Éxito',
    loading: 'Cargando...'
  }
}
