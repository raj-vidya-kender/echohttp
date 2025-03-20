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

  const formatData = (data: any): string => {
    if (typeof data === 'string') {
      try {
        // Try to parse as JSON
        const parsed = JSON.parse(data);
        return JSON.stringify(parsed, null, 2);
      } catch {
        // If not JSON, return as is
        return data;
      }
    }
    return JSON.stringify(data, null, 2);
  };

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
                <pre>{formatData(request.data)}</pre>
              </div>
              <div className="headers">
                <h4>Headers</h4>
                <div className="headers-grid">
                  {(() => {
                    const entries = Object.entries(request.headers);
                    const midPoint = Math.ceil(entries.length / 2);
                    const leftHeaders = entries.slice(0, midPoint);
                    const rightHeaders = entries.slice(midPoint);
                    return (
                      <div className="headers-columns">
                        <div className="headers-column">
                          {leftHeaders.map(([key, values]) => (
                            <div key={key} className="header-row">
                              <div className="header-key">{key}</div>
                              <div className="header-value">{values.join(', ')}</div>
                            </div>
                          ))}
                        </div>
                        <div className="headers-column">
                          {rightHeaders.map(([key, values]) => (
                            <div key={key} className="header-row">
                              <div className="header-key">{key}</div>
                              <div className="header-value">{values.join(', ')}</div>
                            </div>
                          ))}
                        </div>
                      </div>
                    );
                  })()}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
