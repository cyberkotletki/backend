class DonlyMiniApp {
    constructor() {
        this.tg = window.Telegram.WebApp;
        this.apiUrl = window.location.origin + '/api';
        this.authToken = null;
        this.init();
    }

    init() {
        console.log('Initializing Donly Mini App...');
        console.log('Telegram WebApp data:', this.tg.initData);

        // Настраиваем Telegram WebApp
        this.tg.ready();
        this.tg.expand();

        // Применяем тему Telegram
        this.applyTelegramTheme();

        // Инициализируем интерфейс
        this.initUI();

        // Проверяем авторизацию
        this.checkAuth();
    }

    applyTelegramTheme() {
        const themeParams = this.tg.themeParams;
        if (themeParams) {
            document.documentElement.style.setProperty('--tg-theme-bg-color', themeParams.bg_color);
            document.documentElement.style.setProperty('--tg-theme-text-color', themeParams.text_color);
            document.documentElement.style.setProperty('--tg-theme-button-color', themeParams.button_color);
            document.documentElement.style.setProperty('--tg-theme-button-text-color', themeParams.button_text_color);
            document.documentElement.style.setProperty('--tg-theme-secondary-bg-color', themeParams.secondary_bg_color);
        }
    }

    initUI() {
        const authBtn = document.getElementById('auth-btn');
        const testApiBtn = document.getElementById('test-api-btn');

        authBtn.addEventListener('click', () => this.authenticate());
        testApiBtn.addEventListener('click', () => this.testAPI());

        // Настраиваем главную кнопку Telegram
        this.tg.MainButton.text = 'Авторизоваться';
        this.tg.MainButton.onClick(() => this.authenticate());
    }

    async checkAuth() {
        try {
            this.hideLoading();

            // Проверяем, есть ли данные от Telegram
            if (!this.tg.initDataUnsafe || !this.tg.initDataUnsafe.user) {
                this.showError('Приложение должно запускаться из Telegram');
                return;
            }

            const user = this.tg.initDataUnsafe.user;
            this.displayUserInfo(user, false);

            // Показываем кнопку авторизации
            document.getElementById('auth-btn').style.display = 'block';
            this.tg.MainButton.show();

        } catch (error) {
            this.showError('Ошибка инициализации: ' + error.message);
        }
    }

    async authenticate() {
        try {
            this.showLoading();

            // Получаем initData для отправки на backend
            const initData = this.tg.initData;

            if (!initData) {
                throw new Error('Нет данных авторизации от Telegram');
            }

            // Отправляем запрос на backend
            const response = await fetch(`${this.apiUrl}/auth/telegram`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    init_data: initData
                })
            });

            if (!response.ok) {
                throw new Error(`Ошибка авторизации: ${response.status}`);
            }

            const data = await response.json();
            this.authToken = data.token;

            this.showSuccess('Авторизация успешна!');
            this.displayUserInfo(this.tg.initDataUnsafe.user, true);

            // Показываем кнопку тестирования API
            document.getElementById('test-api-btn').style.display = 'block';

            // Обновляем главную кнопку
            this.tg.MainButton.text = 'Тестировать API';
            this.tg.MainButton.onClick(() => this.testAPI());

        } catch (error) {
            this.showError('Ошибка авторизации: ' + error.message);
        } finally {
            this.hideLoading();
        }
    }

    async testAPI() {
        try {
            this.showLoading();

            if (!this.authToken) {
                throw new Error('Необходима авторизация');
            }

            const response = await fetch(`${this.apiUrl}/profile`, {
                method: 'GET',
                headers: {
                    'Authorization': `Bearer ${this.authToken}`,
                    'Content-Type': 'application/json',
                }
            });

            if (!response.ok) {
                throw new Error(`Ошибка API: ${response.status}`);
            }

            const data = await response.json();
            this.displayApiResult(data);

        } catch (error) {
            this.showError('Ошибка API: ' + error.message);
        } finally {
            this.hideLoading();
        }
    }

    displayUserInfo(user, isAuthenticated) {
        const userInfo = document.getElementById('user-info');
        const userId = document.getElementById('user-id');
        const userName = document.getElementById('user-name');
        const userUsername = document.getElementById('user-username');
        const authStatus = document.getElementById('auth-status');

        userId.textContent = user.id;
        userName.textContent = `${user.first_name} ${user.last_name || ''}`.trim();
        userUsername.textContent = user.username || 'Не указан';
        authStatus.textContent = isAuthenticated ? 'Авторизован' : 'Не авторизован';

        userInfo.style.display = 'block';
    }

    displayApiResult(data) {
        const apiResult = document.getElementById('api-result');
        const apiResponse = document.getElementById('api-response');

        apiResponse.textContent = JSON.stringify(data, null, 2);
        apiResult.style.display = 'block';
    }

    showLoading() {
        document.getElementById('loading').style.display = 'block';
    }

    hideLoading() {
        document.getElementById('loading').style.display = 'none';
    }

    showError(message) {
        const errorDiv = document.getElementById('error');
        errorDiv.textContent = message;
        errorDiv.style.display = 'block';

        setTimeout(() => {
            errorDiv.style.display = 'none';
        }, 5000);
    }

    showSuccess(message) {
        const successDiv = document.getElementById('success');
        successDiv.textContent = message;
        successDiv.style.display = 'block';

        setTimeout(() => {
            successDiv.style.display = 'none';
        }, 3000);
    }
}

// Инициализируем приложение после загрузки DOM
document.addEventListener('DOMContentLoaded', () => {
    new DonlyMiniApp();
});
