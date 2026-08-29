import './global.css';

import { useEffect, useState } from 'react';
import {
  Linking,
  StatusBar,
  StyleSheet,
  useColorScheme,
  View,
} from 'react-native';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { GestureHandlerRootView } from 'react-native-gesture-handler';
import { NavigationContainer } from '@react-navigation/native';
import { AuthProvider, useAuth } from './src/context/AuthContext';
import { AssignmentsProvider } from './src/context/AssignmentsContext';
import { SubmissionsProvider } from './src/context/SubmissionsContext';
import { SplashScreen } from './src/screens/SplashScreen';
import { LoginScreen } from './src/screens/LoginScreen';
import { RegisterScreen } from './src/screens/RegisterScreen';
import { HomeScreen } from './src/screens/HomeScreen';
import { BottomNavBar } from './src/components/BottomNavBar';
import { TasksStackNavigator } from './src/navigation/TasksStackNavigator';

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
  const { user, isBootstrapping, loginWithToken } = useAuth();
  const [authMode, setAuthMode] = useState<'login' | 'register'>('login');

  // Handles the redirect back from the system browser once a student
  // approves Google sign-in (gradleapp://auth-callback?token=...), both for
  // a cold start (app was closed) and while already running.
  useEffect(() => {
    function handleUrl(url: string) {
      // Avoid relying on the URL global for a custom (non-http) scheme —
      // just pull the token param out with a regex instead.
      const match = url.match(/[?&]token=([^&]+)/);
      if (match) {
        loginWithToken(decodeURIComponent(match[1])).catch(() => {});
      }
    }

    Linking.getInitialURL().then(url => {
      if (url) handleUrl(url);
    });
    const subscription = Linking.addEventListener('url', ({ url }) =>
      handleUrl(url),
    );
    return () => subscription.remove();
  }, [loginWithToken]);

  if (isBootstrapping) return <SplashScreen />;
  if (!user) {
    return authMode === 'login' ? (
      <LoginScreen onSwitchToRegister={() => setAuthMode('register')} />
    ) : (
      <RegisterScreen onSwitchToLogin={() => setAuthMode('login')} />
    );
  }

  return (
    <AssignmentsProvider>
      <SubmissionsProvider>
        <AuthenticatedApp />
      </SubmissionsProvider>
    </AssignmentsProvider>
  );
}

function AuthenticatedApp() {
  const [activeTab, setActiveTab] = useState<'tasks' | 'home'>('home');

  return (
    <View style={styles.container}>
      <View style={styles.container}>
        <View style={activeTab === 'home' ? styles.container : styles.hidden}>
          <HomeScreen />
        </View>
        <View style={activeTab === 'tasks' ? styles.container : styles.hidden}>
          <NavigationContainer>
            <TasksStackNavigator />
          </NavigationContainer>
        </View>
      </View>
      <BottomNavBar activeTab={activeTab} onTabPress={setActiveTab} />
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  hidden: { display: 'none' },
});

export default App;
