import React from 'react';

const RPS_STOPS = [5, 10, 20, 40, 70, 100];

const RpsDial = ({ value, onChange, className = '' }) => {
    const current = RPS_STOPS.includes(value) ? value : RPS_STOPS[1];

    return (
        <div className={`rps-dial ${className}`.trim()}>
            {RPS_STOPS.map((stop, index) => {
                const id = `rps-stop-${stop}`;
                const style = { '--index': String(index + 1) };
                return (
                    <React.Fragment key={stop}>
                        <input
                            type="radio"
                            id={id}
                            name="rps-dial"
                            checked={current === stop}
                            onChange={() => onChange(stop)}
                        />
                        <label htmlFor={id} style={style} />
                    </React.Fragment>
                );
            })}
            <div
                className="rps-dial__needle"
                style={{ '--position': String(RPS_STOPS.indexOf(current)) }}
            />
            <div className="rps-dial__notches">
                {RPS_STOPS.map((stop, index) => (
                    <div
                        key={stop}
                        className="rps-dial__notch"
                        style={{ '--index': String(index + 1) }}
                    >
                        <span>{stop}</span>
                    </div>
                ))}
            </div>
        </div>
    );
};

export default RpsDial;
