/**
 * Sample React Native App
 * https://github.com/facebook/react-native
 *
 * @format
 */

import './global.css';

import { useState } from 'react';
import { StatusBar, StyleSheet, useColorScheme, View } from 'react-native';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { useAuth } from './src/context/AuthContext';
import { SplashScreen } from './src/screens/SplashScreen';
import { LoginScreen } from './src/screens/LoginScreen';

import { RegisterScreen } from './src/screens/RegisterScreen';
import { HomeScreen } from './src/screens/HomeScreen';

function App() {
  const isDarkMode = useColorScheme() === 'dark';

  return (
    <SafeAreaProvider>
      <StatusBar barStyle={isDarkMode ? 'light-content' : 'dark-content'} />
      <AppContent />
    </SafeAreaProvider>
  );
}

function AppContent() {
  const { user, isBootstrapping } = useAuth();
  const [authMode, setAuthMode] = useState<'login' | 'register'>('login');

  if (isBootstrapping) return <SplashScreen />;
  if (!user) {
    return authMode === 'login' ? (
      <LoginScreen onSwitchToRegister={() => setAuthMode('register')} />
    ) : (
      <RegisterScreen onSwitchToLogin={() => setAuthMode('login')} />
    );
  }

  return <HomeScreen />;
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
});

export default App;
