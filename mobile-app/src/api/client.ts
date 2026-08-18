import axios from 'axios';
import { Platform } from 'react-native';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { TOKEN_STORAGE_KEY } from './tokenStorage';

const API_BASE_URL = Platform.select({
  ios: 'http://localhost:8080',
  android: 'http://10.0.2.2:8080',
  default: 'http://localhost:8080',
});

export const apiClient = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
});

apiClient.interceptors.request.use(async config=>{
    const token = await AsyncStorage.getItem(TOKEN_STORAGE_KEY);
    if(token){
        config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
});