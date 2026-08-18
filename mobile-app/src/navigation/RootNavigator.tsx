import { useAuth } from '../context/AuthContext';
import { SplashScreen } from '../screens/SplashScreen';
import { AuthNavigator } from './AuthNavigator';
import { MainNavigator } from './MainNavigator';

export function RootNavigator() {
  const { user, isBootstrapping } = useAuth();

  if (isBootstrapping) {
    return <SplashScreen />;
  }

  return user ? <MainNavigator /> : <AuthNavigator />;
}
