import { useEffect, useState } from 'react';

interface RequestData {
  timestamp: string;
  data: any;
  headers: Record<string, string[]>;
}

export function RequestList() {
  const [requests, setRequests] = useState<RequestData[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchRequests = async () => {
      try {
        const response = await fetch('/echo');
        if (!response.ok) {
          throw new Error('Failed to fetch requests');
        }
        const data = await response.json();
        setRequests(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'An error occurred');
      }
    };

    fetchRequests();
    // Set up polling every 5 seconds
    const interval = setInterval(fetchRequests, 5000);
    return () => clearInterval(interval);
  }, []);

  if (error) {
    return <div className="error">Error: {error}</div>;
  }

  return (
    <div className="request-list">
      {requests.length === 0 ? (
        <p>No requests received yet</p>
      ) : (
        <div className="requests">
          {requests.map((request, index) => (
            <div key={index} className="request-item">
              <div className="timestamp">
                {new Date(request.timestamp).toLocaleString()}
              </div>
              <div className="data">
                <pre>{JSON.stringify(request.data, null, 2)}</pre>
              </div>
              <div className="headers">
                <h4>Headers:</h4>
                <pre>{JSON.stringify(request.headers, null, 2)}</pre>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
