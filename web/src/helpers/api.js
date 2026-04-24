/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import {
  getUserIdFromLocalStorage,
  showError,
  formatMessageForAPI,
  getTextContent,
  isValidMessage,
} from './utils';
import axios from 'axios';
import { MESSAGE_ROLES } from '../constants/playground.constants';

const PUBLIC_SIGNED_ROUTES = new Set([
  '/api/verification',
  '/api/user/register',
  '/api/user/login',
]);

let cachedPublicRequestSigningKey = null;
let publicRequestSigningKeyPromise = null;

export let API = axios.create({
  baseURL: import.meta.env.VITE_REACT_APP_SERVER_URL
    ? import.meta.env.VITE_REACT_APP_SERVER_URL
    : '',
  headers: {
    'New-API-User': getUserIdFromLocalStorage(),
    'Cache-Control': 'no-store',
  },
});

function getBaseURL() {
  return import.meta.env.VITE_REACT_APP_SERVER_URL
    ? import.meta.env.VITE_REACT_APP_SERVER_URL
    : window.location.origin;
}

function getStoredStatus() {
  try {
    const raw = localStorage.getItem('status');
    return raw ? JSON.parse(raw) : null;
  } catch (error) {
    return null;
  }
}

function storeStatus(status) {
  if (!status) {
    return;
  }
  localStorage.setItem('status', JSON.stringify(status));
}

async function refreshPublicRequestSigningKey() {
  if (!publicRequestSigningKeyPromise) {
    publicRequestSigningKeyPromise = axios
      .get('/api/status', {
        baseURL: getBaseURL(),
        headers: {
          'Cache-Control': 'no-store',
        },
      })
      .then((res) => {
        const status = res?.data?.data || null;
        if (status) {
          storeStatus(status);
        }
        const key = status?.public_request_signing_key || '';
        if (key) {
          cachedPublicRequestSigningKey = key;
        }
        return key;
      })
      .finally(() => {
        publicRequestSigningKeyPromise = null;
      });
  }

  return publicRequestSigningKeyPromise;
}

async function getPublicRequestSigningKey() {
  const refreshedKey = await refreshPublicRequestSigningKey();
  if (refreshedKey) {
    return refreshedKey;
  }

  if (cachedPublicRequestSigningKey) {
    return cachedPublicRequestSigningKey;
  }

  const storedStatus = getStoredStatus();
  const storedKey = storedStatus?.public_request_signing_key;
  if (storedKey) {
    cachedPublicRequestSigningKey = storedKey;
    return storedKey;
  }

  return '';
}

function getRequestPathWithQuery(config) {
  const base = config.baseURL || getBaseURL();
  const url = new URL(config.url, base);
  if (config.params) {
    Object.entries(config.params).forEach(([key, value]) => {
      if (value === undefined || value === null) {
        return;
      }
      if (Array.isArray(value)) {
        value.forEach((item) => url.searchParams.append(key, item));
        return;
      }
      url.searchParams.append(key, value);
    });
  }
  return `${url.pathname}${url.search}`;
}

function shouldSignRequest(config) {
  if (!config?.url) {
    return false;
  }
  const target = getRequestPathWithQuery(config);
  const url = new URL(target, window.location.origin);
  return PUBLIC_SIGNED_ROUTES.has(url.pathname);
}

function normalizeRequestBody(data) {
  if (data === undefined || data === null) {
    return '';
  }
  if (typeof data === 'string') {
    return data;
  }
  if (data instanceof URLSearchParams) {
    return data.toString();
  }
  if (typeof FormData !== 'undefined' && data instanceof FormData) {
    const params = new URLSearchParams();
    data.forEach((value, key) => {
      params.append(key, typeof value === 'string' ? value : String(value));
    });
    return params.toString();
  }
  return JSON.stringify(data);
}

function buildPublicRequestSignaturePayload(method, target, timestamp, body) {
  return [method.toUpperCase(), target, String(timestamp), body].join('\n');
}

async function hmacSha256Hex(key, data) {
  const encoder = new TextEncoder();
  const cryptoKey = await window.crypto.subtle.importKey(
    'raw',
    encoder.encode(key),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign'],
  );
  const signature = await window.crypto.subtle.sign(
    'HMAC',
    cryptoKey,
    encoder.encode(data),
  );
  return Array.from(new Uint8Array(signature))
    .map((byte) => byte.toString(16).padStart(2, '0'))
    .join('');
}

function redirectToOAuthUrl(url, options = {}) {
  const { openInNewTab = false } = options;
  const targetUrl = typeof url === 'string' ? url : url.toString();

  if (openInNewTab) {
    window.open(targetUrl, '_blank');
    return;
  }

  window.location.assign(targetUrl);
}

function patchAPIInstance(instance) {
  const originalGet = instance.get.bind(instance);
  const inFlightGetRequests = new Map();

  const genKey = (url, config = {}) => {
    const params = config.params ? JSON.stringify(config.params) : '{}';
    return `${url}?${params}`;
  };

  instance.get = (url, config = {}) => {
    if (config?.disableDuplicate) {
      return originalGet(url, config);
    }

    const key = genKey(url, config);
    if (inFlightGetRequests.has(key)) {
      return inFlightGetRequests.get(key);
    }

    const reqPromise = originalGet(url, config).finally(() => {
      inFlightGetRequests.delete(key);
    });

    inFlightGetRequests.set(key, reqPromise);
    return reqPromise;
  };

  instance.interceptors.request.use(async (config) => {
    if (!shouldSignRequest(config)) {
      return config;
    }

    const signingKey = await getPublicRequestSigningKey();
    if (!signingKey) {
      throw new Error('Failed to load public request signing key');
    }

    const timestamp = Math.floor(Date.now() / 1000);
    const target = getRequestPathWithQuery(config);
    const body = normalizeRequestBody(config.data);
    const payload = buildPublicRequestSignaturePayload(
      config.method || 'GET',
      target,
      timestamp,
      body,
    );
    const signature = await hmacSha256Hex(signingKey, payload);

    config.headers = config.headers || {};
    config.headers['X-NewAPI-Timestamp'] = String(timestamp);
    config.headers['X-NewAPI-Signature'] = signature;
    return config;
  });

  instance.interceptors.response.use(
    (response) => response,
    (error) => {
      if (error.config && error.config.skipErrorHandler) {
        return Promise.reject(error);
      }
      showError(error);
      return Promise.reject(error);
    },
  );
}

patchAPIInstance(API);

export function updateAPI() {
  API = axios.create({
    baseURL: import.meta.env.VITE_REACT_APP_SERVER_URL
      ? import.meta.env.VITE_REACT_APP_SERVER_URL
      : '',
    headers: {
      'New-API-User': getUserIdFromLocalStorage(),
      'Cache-Control': 'no-store',
    },
  });

  patchAPIInstance(API);
}

// playground

export const isImageGenerationModel = (model = '') => {
  const normalized = String(model || '').toLowerCase();
  return (
    normalized.startsWith('gpt-image-') ||
    normalized.startsWith('dall-e') ||
    normalized.startsWith('chatgpt-image') ||
    normalized.startsWith('grok-imagine-image')
  );
};

// 构建API请求负载
export const buildApiPayload = (
  messages,
  systemPrompt,
  inputs,
  parameterEnabled,
) => {
  const processedMessages = messages
    .filter(isValidMessage)
    .map(formatMessageForAPI)
    .filter(Boolean);

  // 如果有系统提示，插入到消息开头
  if (systemPrompt && systemPrompt.trim()) {
    processedMessages.unshift({
      role: MESSAGE_ROLES.SYSTEM,
      content: systemPrompt.trim(),
    });
  }

  const payload = {
    model: inputs.model,
    group: String(inputs.group || '').trim(),
    messages: processedMessages,
    stream: inputs.stream,
  };

  // 添加启用的参数
  const parameterMappings = {
    temperature: 'temperature',
    top_p: 'top_p',
    max_tokens: 'max_tokens',
    frequency_penalty: 'frequency_penalty',
    presence_penalty: 'presence_penalty',
    seed: 'seed',
  };

  Object.entries(parameterMappings).forEach(([key, param]) => {
    const enabled = parameterEnabled[key];
    const value = inputs[param];
    const hasValue = value !== undefined && value !== null;

    if (!enabled) {
      return;
    }

    if (param === 'max_tokens') {
      if (typeof value === 'number') {
        payload[param] = value;
      }
      return;
    }

    if (hasValue) {
      payload[param] = value;
    }
  });

  return payload;
};

export const buildImageGenerationPayload = (messages, inputs) => {
  const lastUserMessage = [...messages]
    .reverse()
    .find((message) => message?.role === MESSAGE_ROLES.USER);
  const prompt = getTextContent(lastUserMessage).trim();

  return {
    model: inputs.model,
    group: String(inputs.group || '').trim(),
    prompt,
    n: 1,
  };
};

export const formatImageGenerationResponse = (data) => {
  const images = Array.isArray(data?.data) ? data.data : [];
  if (images.length === 0) {
    return (
      '图片生成完成，但响应中没有图片数据。\n\n```json\n' +
      JSON.stringify(data, null, 2) +
      '\n```'
    );
  }

  return images
    .map((item, index) => {
      const title = `生成图片 ${index + 1}`;
      const imageUrl =
        item.url ||
        (item.b64_json ? `data:image/png;base64,${item.b64_json}` : '');
      const revisedPrompt = item.revised_prompt
        ? `\n\n> ${item.revised_prompt}`
        : '';
      if (!imageUrl) {
        return `${title}\n\n\`\`\`json\n${JSON.stringify(item, null, 2)}\n\`\`\``;
      }
      return `![${title}](${imageUrl})${revisedPrompt}`;
    })
    .join('\n\n');
};

// 处理API错误响应
export const handleApiError = (error, response = null) => {
  const errorInfo = {
    error: error.message || '未知错误',
    timestamp: new Date().toISOString(),
    stack: error.stack,
  };

  if (response) {
    errorInfo.status = response.status;
    errorInfo.statusText = response.statusText;
  }

  if (error.message.includes('HTTP error')) {
    errorInfo.details = '服务器返回了错误状态码';
  } else if (error.message.includes('Failed to fetch')) {
    errorInfo.details = '网络连接失败或服务器无响应';
  }

  return errorInfo;
};

// 处理模型数据
export const processModelsData = (data, currentModel) => {
  const modelOptions = data.map((model) => ({
    label: model,
    value: model,
  }));

  const hasCurrentModel = modelOptions.some(
    (option) => option.value === currentModel,
  );
  const selectedModel =
    hasCurrentModel && modelOptions.length > 0
      ? currentModel
      : modelOptions[0]?.value;

  return { modelOptions, selectedModel };
};

// 处理分组数据
export const processGroupsData = (data, userGroup) => {
  let groupOptions = Object.entries(data).map(([group, info]) => ({
    label:
      info.desc.length > 20 ? info.desc.substring(0, 20) + '...' : info.desc,
    value: group,
    ratio: info.ratio,
    fullLabel: info.desc,
  }));

  if (groupOptions.length === 0) {
    groupOptions = [
      {
        label: '用户分组',
        value: '',
        ratio: 1,
      },
    ];
  } else if (userGroup) {
    const userGroupIndex = groupOptions.findIndex((g) => g.value === userGroup);
    if (userGroupIndex > -1) {
      const userGroupOption = groupOptions.splice(userGroupIndex, 1)[0];
      groupOptions.unshift(userGroupOption);
    }
  }

  return groupOptions;
};

// 原来components中的utils.js

export async function getOAuthState() {
  let path = '/api/oauth/state';
  let affCode = localStorage.getItem('aff');
  if (affCode && affCode.length > 0) {
    path += `?aff=${affCode}`;
  }
  const res = await API.get(path);
  const { success, message, data } = res.data;
  if (success) {
    return data;
  } else {
    showError(message);
    return '';
  }
}

async function prepareOAuthState(options = {}) {
  const { shouldLogout = false } = options;
  if (shouldLogout) {
    try {
      await API.get('/api/user/logout', { skipErrorHandler: true });
    } catch (err) {}
    localStorage.removeItem('user');
    updateAPI();
  }
  return await getOAuthState();
}

export async function onDiscordOAuthClicked(client_id, options = {}) {
  const state = await prepareOAuthState(options);
  if (!state) return;
  const redirect_uri = `${window.location.origin}/oauth/discord`;
  const response_type = 'code';
  const scope = 'identify+openid';
  redirectToOAuthUrl(
    `https://discord.com/oauth2/authorize?client_id=${client_id}&redirect_uri=${redirect_uri}&response_type=${response_type}&scope=${scope}&state=${state}`,
  );
}

export async function onGoogleOAuthClicked(client_id, options = {}) {
  const state = await prepareOAuthState(options);
  if (!state) return;
  const url = new URL('https://accounts.google.com/o/oauth2/v2/auth');
  url.searchParams.set('client_id', client_id);
  url.searchParams.set('redirect_uri', `${window.location.origin}/oauth/google`);
  url.searchParams.set('response_type', 'code');
  url.searchParams.set('scope', 'openid profile email');
  url.searchParams.set('state', state);
  url.searchParams.set('prompt', 'select_account');
  redirectToOAuthUrl(url);
}

export async function onOIDCClicked(
  auth_url,
  client_id,
  openInNewTab = false,
  options = {},
) {
  const state = await prepareOAuthState(options);
  if (!state) return;
  const url = new URL(auth_url);
  url.searchParams.set('client_id', client_id);
  url.searchParams.set('redirect_uri', `${window.location.origin}/oauth/oidc`);
  url.searchParams.set('response_type', 'code');
  url.searchParams.set('scope', 'openid profile email');
  url.searchParams.set('state', state);
  redirectToOAuthUrl(url, { openInNewTab });
}

export async function onGitHubOAuthClicked(github_client_id, options = {}) {
  const state = await prepareOAuthState(options);
  if (!state) return;
  redirectToOAuthUrl(
    `https://github.com/login/oauth/authorize?client_id=${github_client_id}&state=${state}&scope=user:email`,
  );
}

export async function onLinuxDOOAuthClicked(
  linuxdo_client_id,
  options = { shouldLogout: false },
) {
  const state = await prepareOAuthState(options);
  if (!state) return;
  redirectToOAuthUrl(
    `https://connect.linux.do/oauth2/authorize?response_type=code&client_id=${linuxdo_client_id}&state=${state}`,
  );
}

/**
 * Initiate custom OAuth login
 * @param {Object} provider - Custom OAuth provider config from status API
 * @param {string} provider.slug - Provider slug (used for callback URL)
 * @param {string} provider.client_id - OAuth client ID
 * @param {string} provider.authorization_endpoint - Authorization URL
 * @param {string} provider.scopes - OAuth scopes (space-separated)
 * @param {Object} options - Options
 * @param {boolean} options.shouldLogout - Whether to logout first
 */
export async function onCustomOAuthClicked(provider, options = {}) {
  const state = await prepareOAuthState(options);
  if (!state) return;

  try {
    const redirect_uri = `${window.location.origin}/oauth/${provider.slug}`;

    // Check if authorization_endpoint is a full URL or relative path
    let authUrl;
    if (
      provider.authorization_endpoint.startsWith('http://') ||
      provider.authorization_endpoint.startsWith('https://')
    ) {
      authUrl = new URL(provider.authorization_endpoint);
    } else {
      // Relative path - this is a configuration error, show error message
      console.error(
        'Custom OAuth authorization_endpoint must be a full URL:',
        provider.authorization_endpoint,
      );
      showError(
        'OAuth 配置错误：授权端点必须是完整的 URL（以 http:// 或 https:// 开头）',
      );
      return;
    }

    authUrl.searchParams.set('client_id', provider.client_id);
    authUrl.searchParams.set('redirect_uri', redirect_uri);
    authUrl.searchParams.set('response_type', 'code');
    authUrl.searchParams.set(
      'scope',
      provider.scopes || 'openid profile email',
    );
    authUrl.searchParams.set('state', state);

    redirectToOAuthUrl(authUrl);
  } catch (error) {
    console.error('Failed to initiate custom OAuth:', error);
    showError('OAuth 登录失败：' + (error.message || '未知错误'));
  }
}

let channelModels = undefined;
export async function loadChannelModels() {
  const res = await API.get('/api/models');
  const { success, data } = res.data;
  if (!success) {
    return;
  }
  channelModels = data;
  localStorage.setItem('channel_models', JSON.stringify(data));
}

export function getChannelModels(type) {
  if (channelModels !== undefined && type in channelModels) {
    if (!channelModels[type]) {
      return [];
    }
    return channelModels[type];
  }
  let models = localStorage.getItem('channel_models');
  if (!models) {
    return [];
  }
  channelModels = JSON.parse(models);
  if (type in channelModels) {
    return channelModels[type];
  }
  return [];
}
