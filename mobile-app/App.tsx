import './global.css';

import { useState } from 'react';
import { StatusBar, StyleSheet, useColorScheme } from 'react-native';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { AuthProvider, useAuth } from './src/context/AuthContext';
import { SplashScreen } from './src/screens/SplashScreen';
import { LoginScreen } from './src/screens/LoginScreen';
import { RegisterScreen } from './src/screens/RegisterScreen';
import { HomeScreen } from './src/screens/HomeScreen';
import { TasksScreen } from './src/screens/TasksScreen';
import { BottomNavBar } from './src/components/BottomNavBar';
import { GestureHandlerRootView } from 'react-native-gesture-handler';
import { View } from 'react-native';

function App() {
  const isDarkMode = useColorScheme() === 'dark';

  return (
    <GestureHandlerRootView style={styles.container}>
      <SafeAreaProvider>
        <StatusBar barStyle={isDarkMode ? 'light-content' : 'dark-content'} />
        <AuthProvider>
          <AppContent />
        </AuthProvider>
      </SafeAreaProvider>
    </GestureHandlerRootView>
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

  return <AuthenticatedApp />;
}

function AuthenticatedApp() {
  const [activeTab, setActiveTab] = useState<'tasks' | 'home'>('home');

  return (
    <View style={styles.container}>
      <View style={styles.container}>
        {activeTab === 'home' ? <HomeScreen /> : <TasksScreen />}
      </View>
      <BottomNavBar
        activeTab={activeTab}
        onTabPress={setActiveTab}
        onCameraPress={() => {}}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
});

export default App;
