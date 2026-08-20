import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useReducer,
  useRef,
} from 'react';
import {
  Assignment,
  AssignmentDetail,
  fetchAssignmentById,
  fetchAssignments,
} from '../api/assignments';

type FetchStatus = 'idle' | 'loading' | 'loaded' | 'error';

type State = {
  list: Assignment[];
  listStatus: FetchStatus;
  listError: string | null;
  detailsById: Record<string, AssignmentDetail>;
  detailStatusById: Record<string, FetchStatus>;
  detailErrorById: Record<string, string | null>;
};

type Action =
  | { type: 'LIST_LOADING' }
  | { type: 'LIST_LOADED'; payload: Assignment[] }
  | { type: 'LIST_ERROR'; payload: string }
  | { type: 'DETAIL_LOADING'; id: string }
  | { type: 'DETAIL_LOADED'; id: string; payload: AssignmentDetail }
  | { type: 'DETAIL_ERROR'; id: string; payload: string };

const initialState: State = {
  list: [],
  listStatus: 'idle',
  listError: null,
  detailsById: {},
  detailStatusById: {},
  detailErrorById: {},
};

function reducer(state: State, action: Action): State {
  switch (action.type) {
    case 'LIST_LOADING':
      return { ...state, listStatus: 'loading', listError: null };
    case 'LIST_LOADED':
      return { ...state, listStatus: 'loaded', list: action.payload };
    case 'LIST_ERROR':
      return { ...state, listStatus: 'error', listError: action.payload };
    case 'DETAIL_LOADING':
      return {
        ...state,
        detailStatusById: { ...state.detailStatusById, [action.id]: 'loading' },
        detailErrorById: { ...state.detailErrorById, [action.id]: null },
      };
    case 'DETAIL_LOADED':
      return {
        ...state,
        detailsById: { ...state.detailsById, [action.id]: action.payload },
        detailStatusById: { ...state.detailStatusById, [action.id]: 'loaded' },
      };
    case 'DETAIL_ERROR':
      return {
        ...state,
        detailStatusById: { ...state.detailStatusById, [action.id]: 'error' },
        detailErrorById: {
          ...state.detailErrorById,
          [action.id]: action.payload,
        },
      };
    default:
      return state;
  }
}

type AssignmentsContextValue = {
  list: Assignment[];
  listStatus: FetchStatus;
  listError: string | null;
  loadAssignments: (force?: boolean) => Promise<void>;
  detailsById: Record<string, AssignmentDetail>;
  detailStatusById: Record<string, FetchStatus>;
  detailErrorById: Record<string, string | null>;
  loadAssignmentDetail: (id: string, force?: boolean) => Promise<void>;
};

const AssignmentsContext = createContext<AssignmentsContextValue | undefined>(
  undefined,
);

export function AssignmentsProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const [state, dispatch] = useReducer(reducer, initialState);

  const stateRef = useRef(state);
  stateRef.current = state;

  const loadAssignments = useCallback(async (force = false) => {
    const status = stateRef.current.listStatus;
    if (!force && (status === 'loading' || status === 'loaded')) {
      return;
    }
    dispatch({ type: 'LIST_LOADING' });
    try {
      const assignments = await fetchAssignments();
      dispatch({ type: 'LIST_LOADED', payload: assignments });
    } catch {
      dispatch({ type: 'LIST_ERROR', payload: 'Failed to load assignments.' });
    }
  }, []);

  const loadAssignmentDetail = useCallback(
    async (id: string, force = false) => {
      const status = stateRef.current.detailStatusById[id];
      if (!force && (status === 'loading' || status === 'loaded')) {
        return;
      }
      dispatch({ type: 'DETAIL_LOADING', id });
      try {
        const detail = await fetchAssignmentById(id);
        dispatch({ type: 'DETAIL_LOADED', id, payload: detail });
      } catch {
        dispatch({
          type: 'DETAIL_ERROR',
          id,
          payload: 'Failed to load assignment.',
        });
      }
    },
    [],
  );

  const value = useMemo(
    () => ({
      list: state.list,
      listStatus: state.listStatus,
      listError: state.listError,
      loadAssignments,
      detailsById: state.detailsById,
      detailStatusById: state.detailStatusById,
      detailErrorById: state.detailErrorById,
      loadAssignmentDetail,
    }),
    [state, loadAssignments, loadAssignmentDetail],
  );

  return (
    <AssignmentsContext.Provider value={value}>
      {children}
    </AssignmentsContext.Provider>
  );
}

export function useAssignments(): AssignmentsContextValue {
  const context = useContext(AssignmentsContext);
  if (!context) {
    throw new Error(
      'useAssignments must be used within an AssignmentsProvider',
    );
  }
  return context;
}
